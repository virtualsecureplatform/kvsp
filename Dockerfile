FROM nvidia/cuda:11.1.1-devel-ubuntu20.04
#FROM nvidia/cuda:11.2.2-devel-ubuntu20.04

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update && apt-get install -y \
    build-essential git curl software-properties-common openjdk-11-jre \
    cmake clang clang-9 bison flex libreadline-dev \
    gawk tcl-dev libffi-dev graphviz xdot pkg-config python3 libboost-system-dev \
	libboost-python-dev libboost-filesystem-dev zlib1g-dev \
    && rm -rf /var/lib/apt/lists/*

# Install .NET Core SDK
RUN curl -sL https://packages.microsoft.com/config/ubuntu/20.04/packages-microsoft-prod.deb -o packages-microsoft-prod.deb
RUN dpkg -i packages-microsoft-prod.deb
RUN apt-get update && apt-get install -y dotnet-sdk-3.1

# Install sbt
RUN echo "deb https://dl.bintray.com/sbt/debian /" | tee -a /etc/apt/sources.list.d/sbt.list
RUN curl -sL "https://keyserver.ubuntu.com/pks/lookup?op=get&search=0x2EE0EA64E40A89B84B2DF73499E82A75642AC823" | apt-key add
RUN apt-get update && apt-get install -y sbt

# Install Go
RUN add-apt-repository ppa:longsleep/golang-backports
RUN apt-get update && apt-get install -y golang-go
RUN go get github.com/BurntSushi/toml

# Run the build when executing `docker run`
CMD ["bash", "-c", "make -j$(nproc) ENABLE_CUDA=1 CUDACXX=\"/usr/local/cuda/bin/nvcc\" CUDAHOSTCXX=\"/usr/bin/clang-9\""]
