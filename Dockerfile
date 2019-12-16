FROM ubuntu:18.04

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update && apt-get -y upgrade
RUN apt-get install -y build-essential git cmake curl software-properties-common
RUN apt-get install -y libtbb-dev openjdk-8-jre

# Install Go
RUN add-apt-repository ppa:longsleep/golang-backports
RUN apt-get update && apt-get install -y golang-go

# Install .NET Core SDK
RUN curl -sL https://packages.microsoft.com/config/ubuntu/18.04/packages-microsoft-prod.deb -o packages-microsoft-prod.deb
RUN dpkg -i packages-microsoft-prod.deb
RUN apt-get update && apt-get install -y dotnet-sdk-3.1

# Install sbt
RUN echo "deb https://dl.bintray.com/sbt/debian /" | tee -a /etc/apt/sources.list.d/sbt.list
RUN curl -sL "https://keyserver.ubuntu.com/pks/lookup?op=get&search=0x2EE0EA64E40A89B84B2DF73499E82A75642AC823" | apt-key add
RUN apt-get update && apt-get install -y sbt

# Build yosys
RUN git clone https://github.com/cliffordwolf/yosys.git
RUN apt-get install -y tcl-dev libreadline6-dev bison flex libffi-dev
RUN cd yosys && make CONFIG=gcc -j $(( $(grep cpu.cores /proc/cpuinfo | sort -u | sed 's/[^0-9]//g') + 1 )) && make CONFIG=gcc install

# Run the build when executing `docker run`
CMD ["make"]
