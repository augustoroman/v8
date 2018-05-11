# This dockerfile will build v8 as static libraries suitable for being linked
# to the github.com/augustoroman/v8 go library bindings.
#
# The V8_VERSION arg is required and specifies which v8 git version to build.
# The recommended incantation to build this is:
#
#   docker build --build-arg V8_VERSION=6.7.77 --tag augustoroman/v8-lib:6.7.77 .
#

FROM ubuntu:16.04 as builder
# Install the basics we need to compile and install stuff.
RUN apt-get update -qq \
    && apt-get install -y --no-install-recommends \
            ca-certificates build-essential pkg-config git curl python \
    && rm -rf /var/lib/apt/lists/*

# Download the depot_tools and the basic v8 code.
ARG BUILD_DIR=/build/chromium
ADD download_v8.sh download_v8.sh
RUN ./download_v8.sh

# Checkout the specific V8_VERSION we want to build.
ARG V8_VERSION
ADD checkout_v8.sh checkout_v8.sh
RUN ./checkout_v8.sh

# Compile it!
ADD compile_v8.sh compile_v8.sh
RUN ./compile_v8.sh

# Some V8 versions produce thin archives by default.
ADD fatten_archives.sh fatten_archives.sh
RUN ./fatten_archives.sh

# Create a clean docker image with only the v8 libs.
FROM tianon/true as lib
ARG BUILD_DIR=/build/chromium
COPY --from=builder ${BUILD_DIR}/v8/out.gn/lib/obj/*.a /v8/lib/
COPY --from=builder ${BUILD_DIR}/v8/include/ /v8/include/
