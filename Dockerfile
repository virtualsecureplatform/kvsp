FROM nvidia/cuda:13.3.1-devel-ubuntu24.04

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update && apt-get install -y \
    build-essential git curl openjdk-17-jdk-headless \
    cmake clang bison flex libreadline-dev autoconf automake libtool libtool-bin \
    gawk tcl-dev libffi-dev graphviz xdot pkg-config python3 libboost-system-dev \
	libboost-python-dev libboost-filesystem-dev zlib1g-dev gnupg \
    && rm -rf /var/lib/apt/lists/*

# Install the maintained sbt launcher and Go toolchain supplied by Ubuntu 24.04.
RUN apt-get update && apt-get install -y golang-go && \
    curl -fsSL https://raw.githubusercontent.com/paulp/sbt-extras/master/sbt \
        -o /usr/local/bin/sbt && \
    chmod 0755 /usr/local/bin/sbt && \
    rm -rf /var/lib/apt/lists/*

# The source tree is bind-mounted at /build and normally owned by the host
# user. Trust the explicitly mounted repositories so Git-dependent Yosys and
# Go builds retain revision metadata when the image runs as root.
RUN for dir in /build /build/Alexandrite /build/Iyokan /build/Tangor \
        /build/cahp-pearl /build/cahp-rt /build/cahp-ruby /build/cahp-sim \
        /build/llvm-cahp /build/yosys /build/yosys/abc \
        /build/yosys/libs/cxxopts; do \
        git config --system --add safe.directory "$$dir"; \
    done

# Run the build when executing `docker run`
CMD ["bash", "-c", "make -j$(nproc) ENABLE_CUDA=1 CUDACXX=\"/usr/local/cuda/bin/nvcc\" CUDAHOSTCXX=\"/usr/bin/g++\""]
