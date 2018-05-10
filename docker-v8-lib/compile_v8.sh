#!/bin/bash -ex

: "${BUILD_DIR:?BUILD_DIR must be set}"

cd $BUILD_DIR
export PATH="$(pwd)/depot_tools:$PATH"
cd v8

gn gen out.gn/lib --args='
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
ninja -C out.gn/lib v8_libbase v8_libplatform v8_base v8_nosnapshot v8_libsampler v8_init v8_initializers
