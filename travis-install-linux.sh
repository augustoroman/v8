#!/bin/bash -ex
#
# This script is used to download and install the v8 libraries on linux for
# travis-ci.
#

: "${V8_VERSION:?V8_VERSION must be set}"

CHROMIUM_DIR=${HOME}/chromium

mkdir -p ${CHROMIUM_DIR}
pushd ${CHROMIUM_DIR}
git clone https://chromium.googlesource.com/chromium/tools/depot_tools.git
export PATH="$(pwd)/depot_tools:$PATH"
gclient
fetch v8
cd v8
git checkout ${V8_VERSION}
gclient sync
tools/dev/v8gen.py x64.release -- v8_use_external_startup_data=false v8_enable_i18n_support=false v8_enable_gdbjit=false v8_static_library=true is_component_build=false
ninja -C out.gn/x64.release v8_libbase v8_libplatform v8_base v8_snapshot
popd
./symlink.sh ${CHROMIUM_DIR}/v8
go install .
