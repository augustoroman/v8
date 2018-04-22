#!/bin/bash -ex

: "${V8_VERSION:?V8_VERSION must be set}"
: "${CHROMIUM_DIR:?CHROMIUM_DIR must be set}"

mkdir -p ${CHROMIUM_DIR}
cd ${CHROMIUM_DIR}
git clone https://chromium.googlesource.com/chromium/tools/depot_tools.git
export PATH="$(pwd)/depot_tools:$PATH"
fetch v8
cd v8
git checkout ${V8_VERSION}
gclient sync
