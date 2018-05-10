ARG V8_VERSION
ARG V8_SOURCE_IMAGE=augustoroman/v8-lib

# ------------ Import the v8 libraries --------------------------------------
# The v8 library & include files are taken from a pre-built docker image that
# is expected to be called v8-lib. You can build that locally using:
#   docker build --build-arg V8_VERSION=6.7.77 --tag augustoroman/v8-lib:6.7.77 docker-v8-lib/
# or you can use a previously built image from:
#   https://hub.docker.com/r/augustoroman/v8-lib/
#
# Once that is available, build this docker image using:
#   docker build --build-arg V8_VERSION=6.7.77 -t v8-runjs .
# and then run the interactive js using:
#   docker run -it --rm v8-runjs
FROM ${V8_SOURCE_IMAGE}:${V8_VERSION} as v8

# ------------ Build go v8 library and run tests ----------------------------
FROM golang as builder
# Copy the v8 code from the local disk, similar to:
#   RUN go get github.com/augustoroman/v8 ||:
# but this allows using any local modifications.
ARG GO_V8_DIR=/go/src/github.com/augustoroman/v8/
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

# ------------ Build the final container for v8-runjs -----------------------
# TODO(aroman) find a smaller container for the executable! For some reason,
# scratch, alpine, and busybox don't work. I wonder if it has something to do
# with cgo?
FROM ubuntu:16.04
COPY --from=builder /go/bin/v8-runjs /v8-runjs
CMD /v8-runjs
