FROM ubuntu:16.04

# ------------ Install and build V8 ---------------------------------

# Install the basics we need to compile and install stuff:
RUN apt-get update -qq && apt-get install -y build-essential pkg-config git curl python

# Now build V8
ENV CHROMIUM_DIR=$HOME/chromium
ENV V8_VERSION=6.7.77
ADD docker-scripts/download_v8.sh download_v8.sh
RUN ./download_v8.sh

ADD docker-scripts/compile_v8.sh compile_v8.sh
RUN ./compile_v8.sh

# ------------ Install go ---------------------------------

RUN curl -OJL https://dl.google.com/go/go1.10.1.linux-amd64.tar.gz
RUN tar -C /usr/local -xzf go1.10.1.linux-amd64.tar.gz
ENV PATH=$PATH:/usr/local/go/bin

# ------------ Build and run v8 go tests ---------------------------------
# Get the v8 code, similar to:
#   RUN go get github.com/augustoroman/v8 ||:
ARG GO_V8_DIR=/root/go/src/github.com/augustoroman/v8/
ADD *.go $GO_V8_DIR
ADD symlink.sh $GO_V8_DIR
ADD *.h $GO_V8_DIR
ADD *.cc $GO_V8_DIR

# Install the library and run tests.
ADD docker-scripts/install_golib.sh install_golib.sh
RUN ./install_golib.sh
