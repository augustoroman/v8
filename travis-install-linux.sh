#!/bin/bash -ex
#
# This script is used to download and install the v8 libraries on linux for
# travis-ci.
#

CHROMIUM_DIR=${HOME}/chromium

mkdir -p ${CHROMIUM_DIR}
pushd ${CHROMIUM_DIR}
git clone https://chromium.googlesource.com/chromium/tools/depot_tools.git
export PATH="$(pwd)/depot_tools:$PATH"
gclient
fetch v8
cd v8
git checkout 5.6.326.18
gclient sync
make x64.release GYPFLAGS="-Dv8_use_external_startup_data=0 -Dv8_enable_i18n_support=0 -Dv8_enable_gdbjit=0"
popd
./symlink.sh ${CHROMIUM_DIR}/v8
go install .
