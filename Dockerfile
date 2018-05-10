ARG V8_VERSION
ARG V8_SOURCE_IMAGE=augustoroman/v8-lib

# The v8 library & include files are taken from a pre-built docker image that
# is expected to be called v8-lib. You can build that locally using:
#   docker build --build-arg V8_VERSION=6.7.77 --tag augustoroman/v8-lib:6.7.77 docker-v8-lib/
# or you can use a previously built image from:
#   https://hub.docker.com/r/augustoroman/v8-lib/
#
# Once that is available, build this docker image using:
#   docker build --build-arg V8_VERSION=6.7.77 -t gov8 .
# and then run the interactive js using:
#   docker run -it --rm gov8
FROM ${V8_SOURCE_IMAGE}:${V8_VERSION} as v8

FROM ubuntu:16.04
# Install the basics we need to compile and install stuff.
# In particular, we'll need curl for installing go and build-essential for
# gcc.
RUN apt-get update -qq \
    && apt-get install -y --no-install-recommends ca-certificates curl build-essential git \
    && rm -rf /var/lib/apt/lists/*

# Download and install go.
RUN curl -JL https://dl.google.com/go/go1.10.2.linux-amd64.tar.gz \
    | tar -C /usr/local -xz
ENV PATH=$PATH:/usr/local/go/bin

# ------------ Build and run v8 go tests ---------------------------------
# Copy the v8 code from the local disk, similar to:
#   RUN go get github.com/augustoroman/v8 ||:
# but this allows using any local modifications.
ARG GO_V8_DIR=/root/go/src/github.com/augustoroman/v8/
ADD *.go *.h *.cc $GO_V8_DIR
ADD cmd $GO_V8_DIR/cmd/
ADD v8console $GO_V8_DIR/v8console/

# Copy the pre-compiled library & include files for the desired v8 version.
COPY --from=v8 /v8/lib $GO_V8_DIR/libv8/
COPY --from=v8 /v8/include $GO_V8_DIR/include/

# Install the go code and run tests.
WORKDIR $GO_V8_DIR
RUN go get ./...
RUN go test ./...

ENTRYPOINT /root/go/bin/v8-runjs
