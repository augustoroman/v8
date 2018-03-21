#!/bin/bash -e

if [[ "$1" == "-h" || -z "$1" ]]; then
  echo "Usage: `basename $0` /path/to/chromium/v8"
  echo ""
  echo "This will create symlinks for the libv8/ and include/ directories necessary"
  echo "to build the v8 Go package.  The path should be the v8 directory with the"
  echo "compiled libraries (see build instructions)."
  exit 0
fi

PKG_DIR=`dirname $0`
V8_DIR=${1%/}
cd ${PKG_DIR}

# Make sure that the specified include dir exists.  This could happen if you
# specify a relative directory that isn't right after cd'ing to PKG_DIR.
if [[ ! -d "${V8_DIR}/include" ]]; then
    echo "ERROR: ${V8_DIR}/include does not exist." >&2
    exit 1
fi

V8_LIBS="out.gn/golib/obj"

if [[ ! -d "${V8_DIR}/${V8_LIBS}" ]]; then
    echo "ERROR: ${V8_DIR}/${V8_LIBS} directory does not exist." >&2
    exit 1
fi

set -x +e
ln -s ${V8_DIR}/${V8_LIBS} libv8
ln -s ${V8_DIR}/include include
