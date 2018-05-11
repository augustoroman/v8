#!/bin/bash -x

: "${BUILD_DIR:?BUILD_DIR must be set}"

cd $BUILD_DIR
export PATH="$(pwd)/depot_tools:$PATH"
cd v8

cd out.gn/lib/obj

# Convert any thin archives into fat ones. This will attempt to
# fatten all archives, but we ignore failures if it's already fat
# via the ||: at the end.
for lib in `find . -name '*.a'`; do
  ar -t $lib | xargs ar rvs $lib.new && mv -v $lib.new $lib ||:
done
