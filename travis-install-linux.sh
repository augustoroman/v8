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
fetch v8
cd v8
git checkout ${V8_VERSION}
gclient sync
gn gen out.gn/golib --args='
    target_cpu = "x64"
    is_debug = false

    symbol_level = 0
    strip_debug_info = true
    v8_experimental_extra_library_files = []
    v8_extra_library_files = []

    v8_static_library = true
    is_component_build = false
    use_custom_libcxx = false
    use_custom_libcxx_for_host = false

    icu_use_data_file = false
    is_desktop_linux = false
    v8_enable_i18n_support = false
    v8_use_external_startup_data = false
    v8_enable_gdbjit = false'
ninja -C out.gn/golib v8_libbase v8_libplatform v8_base v8_nosnapshot v8_libsampler v8_init v8_initializers
popd
./symlink.sh ${CHROMIUM_DIR}/v8
go install .
