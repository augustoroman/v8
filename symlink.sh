#!/bin/bash -e

if [[ "$1" == "-h" || -z "$1" ]]; then
  echo "Usage: `basename $0` /path/to/chromium/v8"
  echo ""
  echo "This will create symlinks for the libv8/ and include/ directories necessary"
  echo "to build the v8 Go package.  The path should be the v8 directory with the"
  echo "compiled libraries (see build instructions)."
  exit 0
fi

V8=${1%/}
set -x +e
ln -s ${V8}/out/x64.release libv8
ln -s ${V8}/include include
