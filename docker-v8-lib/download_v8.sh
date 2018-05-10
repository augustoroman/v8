#!/bin/bash -ex

: "${BUILD_DIR:?BUILD_DIR must be set}"

mkdir -p ${BUILD_DIR}
cd ${BUILD_DIR}
git clone https://chromium.googlesource.com/chromium/tools/depot_tools.git
export PATH="$(pwd)/depot_tools:$PATH"
fetch v8
