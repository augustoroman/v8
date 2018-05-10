#!/bin/bash -ex
#
# This script is used to download and install the v8 libraries on linux for
# travis-ci.
#

: "${V8_VERSION:?V8_VERSION must be set}"

V8_DIR=${HOME}/libv8gem
mkdir -p ${V8_DIR}
pushd ${V8_DIR}

curl https://rubygems.org/downloads/libv8-${V8_VERSION}-x86_64-linux.gem | tar xv
tar xzvf data.tar.gz

popd

ln -s ${V8_DIR}/vendor/v8/out/x64.release libv8
ln -s ${V8_DIR}/vendor/v8/include include

go get ./...
