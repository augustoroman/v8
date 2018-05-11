#!/bin/bash -ex

: "${V8_VERSION:?V8_VERSION must be set}"
: "${BUILD_DIR:?BUILD_DIR must be set}"

cd $BUILD_DIR
export PATH="$(pwd)/depot_tools:$PATH"
cd v8

git fetch origin ${V8_VERSION}
git checkout ${V8_VERSION}
gclient sync
