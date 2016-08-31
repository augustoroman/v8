#!/bin/bash -ex

mkdir ${HOME}/chromium
pushd ${HOME}/chromium
git clone https://chromium.googlesource.com/chromium/tools/depot_tools.git
export PATH="$(pwd)/depot_tools:$PATH"
gclient
fetch v8
cd v8
git checkout 5.4.374.1
gclient sync
make x64.release GYPFLAGS="-Dv8_use_external_startup_data=0 -Dv8_enable_i18n_support=0 -Dv8_enable_gdbjit=0"
popd
./symlink.sh ${HOME}/chromium/v8
go install .
