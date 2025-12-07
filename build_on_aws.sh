#!/bin/bash
set -e

# Build KVSP on AWS c5 spot instance
# Usage: ./build_on_aws.sh <version>
# Example: ./build_on_aws.sh 30

VERSION="$1"
if [ -z "$VERSION" ]; then
    echo "Usage: $0 <version>"
    echo "Example: $0 30"
    exit 1
fi

# Configuration
INSTANCE_TYPE="c5.12xlarge"
AMI_ID="ami-02dfeaf60af852169"  # Ubuntu 24.04 LTS - update for your region
REGION="us-east-2"
KEY_NAME="${AWS_KEY_NAME:-}"
SECURITY_GROUP="${AWS_SECURITY_GROUP:-}" # Must allow SSH
SUBNET_ID="${AWS_SUBNET_ID:-}"
SPOT_PRICE="${SPOT_PRICE:-2.00}"

# Local paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SIF_FILE="$SCRIPT_DIR/kvsp.sif"
DEF_FILE="$SCRIPT_DIR/kvsp.def"

echo "=== KVSP AWS Spot Instance Build Script ==="
echo "Version: $VERSION"
echo "Instance type: $INSTANCE_TYPE"
echo ""

# Step 1: Build kvsp.sif locally if missing
if [ ! -f "$SIF_FILE" ]; then
    echo "=== Building kvsp.sif locally ==="
    if [ ! -f "$DEF_FILE" ]; then
        echo "Error: kvsp.def not found at $DEF_FILE"
        exit 1
    fi

    # Check if apptainer/singularity is available locally
    if command -v apptainer &> /dev/null; then
        apptainer build "$SIF_FILE" "$DEF_FILE"
    elif command -v singularity &> /dev/null; then
        singularity build --fakeroot "$SIF_FILE" "$DEF_FILE"
    else
        echo "Error: Neither apptainer nor singularity is installed locally."
        echo "Please install apptainer to build the container image."
        exit 1
    fi
    echo "=== kvsp.sif built successfully ==="
else
    echo "=== kvsp.sif already exists, skipping local build ==="
fi

# Step 2: Create spot instance request
echo "=== Creating AWS c5 spot instance ==="

# Prepare user data script for instance initialization
USER_DATA=$(cat <<'USERDATA'
#!/bin/bash
set -e
exec > /var/log/user-data.log 2>&1

# Update and install dependencies
apt-get update
apt-get upgrade -y
apt-get install -y libgoogle-perftools-dev libomp-dev pigz

# Install apptainer
apt-get install -y software-properties-common
add-apt-repository -y ppa:apptainer/ppa
apt-get update
apt-get install -y apptainer

# Signal that instance is ready
touch /tmp/instance-ready
USERDATA
)

USER_DATA_B64=$(echo "$USER_DATA" | base64 -w 0)

# Build subnet option if specified
SUBNET_OPT=""
if [ -n "$SUBNET_ID" ]; then
    SUBNET_OPT="SubnetId=$SUBNET_ID,"
fi

# Request spot instance
SPOT_REQUEST=$(aws ec2 request-spot-instances \
    --region "$REGION" \
    --spot-price "$SPOT_PRICE" \
    --instance-count 1 \
    --type "one-time" \
    --launch-specification "{
        \"ImageId\": \"$AMI_ID\",
        \"InstanceType\": \"$INSTANCE_TYPE\",
        \"KeyName\": \"$KEY_NAME\",
        \"SecurityGroupIds\": [\"$SECURITY_GROUP\"],
        ${SUBNET_OPT:+\"SubnetId\": \"$SUBNET_ID\",}
        \"UserData\": \"$USER_DATA_B64\",
        \"BlockDeviceMappings\": [{
            \"DeviceName\": \"/dev/sda1\",
            \"Ebs\": {
                \"VolumeSize\": 50,
                \"VolumeType\": \"gp3\",
                \"DeleteOnTermination\": true
            }
        }]
    }" \
    --output json)

SPOT_REQUEST_ID=$(echo "$SPOT_REQUEST" | jq -r '.SpotInstanceRequests[0].SpotInstanceRequestId')
echo "Spot request ID: $SPOT_REQUEST_ID"

# Wait for spot request to be fulfilled
echo "Waiting for spot request to be fulfilled..."
aws ec2 wait spot-instance-request-fulfilled \
    --region "$REGION" \
    --spot-instance-request-ids "$SPOT_REQUEST_ID"

# Get instance ID
INSTANCE_ID=$(aws ec2 describe-spot-instance-requests \
    --region "$REGION" \
    --spot-instance-request-ids "$SPOT_REQUEST_ID" \
    --query 'SpotInstanceRequests[0].InstanceId' \
    --output text)

echo "Instance ID: $INSTANCE_ID"

# Wait for instance to be running
echo "Waiting for instance to be running..."
aws ec2 wait instance-running \
    --region "$REGION" \
    --instance-ids "$INSTANCE_ID"

# Get public IP
INSTANCE_IP=$(aws ec2 describe-instances \
    --region "$REGION" \
    --instance-ids "$INSTANCE_ID" \
    --query 'Reservations[0].Instances[0].PublicIpAddress' \
    --output text)

echo "Instance IP: $INSTANCE_IP"

# Function to cleanup on exit
cleanup() {
    echo ""
    echo "=== Cleaning up ==="
    if [ -n "$INSTANCE_ID" ]; then
        echo "Terminating instance $INSTANCE_ID..."
        aws ec2 terminate-instances \
            --region "$REGION" \
            --instance-ids "$INSTANCE_ID" || true
        echo "Instance termination requested."
    fi
    if [ -n "$SPOT_REQUEST_ID" ]; then
        aws ec2 cancel-spot-instance-requests \
            --region "$REGION" \
            --spot-instance-request-ids "$SPOT_REQUEST_ID" || true
    fi
}

# Set trap to cleanup on script exit (including errors)
trap cleanup EXIT

# Wait for SSH to be available
echo "Waiting for SSH to be available..."
for i in {1..60}; do
    if ssh -o StrictHostKeyChecking=no -o ConnectTimeout=5 -o BatchMode=yes \
        ubuntu@"$INSTANCE_IP" "echo 'SSH ready'" 2>/dev/null; then
        break
    fi
    echo "  Attempt $i/60..."
    sleep 10
done

# Wait for user-data script to complete
echo "Waiting for instance initialization to complete..."
for i in {1..60}; do
    if ssh -o StrictHostKeyChecking=no ubuntu@"$INSTANCE_IP" \
        "test -f /tmp/instance-ready" 2>/dev/null; then
        echo "Instance initialization complete."
        break
    fi
    echo "  Waiting for initialization... ($i/60)"
    sleep 10
done

# Step 3: Copy kvsp.sif to instance
echo "=== Copying kvsp.sif to instance ==="
scp -o StrictHostKeyChecking=no "$SIF_FILE" ubuntu@"$INSTANCE_IP":~/kvsp.sif

# Step 4: Clone and build on instance
echo "=== Building KVSP on instance ==="
ssh -o StrictHostKeyChecking=no ubuntu@"$INSTANCE_IP" bash -s "$VERSION" <<'REMOTE_SCRIPT'
set -e
VERSION="$1"

echo "=== Cloning KVSP repository ==="
git clone --recursive https://github.com/virtualsecureplatform/kvsp.git
cd kvsp

# Move the sif file into the project
mv ~/kvsp.sif ./kvsp.sif

echo "=== Building KVSP with ENABLE_CUDA=1 ==="
apptainer exec --bind $(pwd):$(pwd) --pwd $(pwd) --writable-tmpfs \
    kvsp.sif make -j24 ENABLE_CUDA=1

echo "=== Running copy command ==="
./toolbox.sh copy "$VERSION"

echo "=== Running pack command ==="
./toolbox.sh pack "$VERSION"

# Rename to versioned tarball
mv kvsp.tar.gz "kvsp_v${VERSION}.tar.gz"

echo "=== Build completed ==="
ls -la "kvsp_v${VERSION}.tar.gz"
REMOTE_SCRIPT

# Step 5: Copy result back to local
echo "=== Copying kvsp_v${VERSION}.tar.gz to local ==="
scp -o StrictHostKeyChecking=no ubuntu@"$INSTANCE_IP":~/kvsp/kvsp_v${VERSION}.tar.gz "$SCRIPT_DIR/"

echo ""
echo "=== Build completed successfully ==="
echo "Output: $SCRIPT_DIR/kvsp_v${VERSION}.tar.gz"
ls -la "$SCRIPT_DIR/kvsp_v${VERSION}.tar.gz"

# Cleanup will be called by trap on exit
echo "=== Terminating instance ==="
