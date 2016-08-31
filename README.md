# V8 Bindings for Go [![Build Status](https://travis-ci.org/augustoroman/v8.svg?branch=master)](https://travis-ci.org/augustoroman/v8)  [![Go Report Card](https://goreportcard.com/badge/github.com/augustoroman/v8)](https://goreportcard.com/report/github.com/augustoroman/v8)  [![GoDoc](https://godoc.org/github.com/augustoroman/v8?status.png)](http://godoc.org/github.com/augustoroman/v8)

The v8 bindings allow a user to execute javascript from within a go executable.

The bindings are tested to work with v8 build 5.4.374.0 (latest dev at the
time of writing).  Note that v8 releases match the Chrome release timeline:
Chrome 48 corresponds to v8 4.8.\*, Chrome 49 matches v8 4.9.\*.  You can see the
table of current chrome and the associated v8 releases at:

  http://omahaproxy.appspot.com/

# Compiling

In order for the bindings to compile correctly, one needs to:

1. Compile v8 as a static library.
2. Let cgo know where the library is located.

## Compiling v8.

Download v8: https://github.com/v8/v8/wiki/Using%20Git

Lets say you've checked out the v8 source into `$V8`, go-v8 into `$GO_V8` and
want to place the static v8 library into `$GO_V8/libv8/`.  (For example,
`export GO_V8=$GOPATH/src/github.com/augustoroman/v8`)


### Linux

Build:

    make x64.release GYPFLAGS="-Dv8_use_external_startup_data=0 -Dv8_enable_i18n_support=0 -Dv8_enable_gdbjit=0"

If build system produces a thin archive, you want to make it into a fat one:

    for lib in `find out/x64.release/obj.target/src/ -name '*.a'`;
      do ar -t $lib | xargs ar rvs $lib.new && mv -v $lib.new $lib;
    done

Symlink the libraries and include directory to the Go package dir:

    ln -s `pwd`/out/x64.release/obj.target/src ${GO_V8}/libv8
    ln -s `pwd`/include ${GO_V8}/include


### Mac

To build: (substitute in your OS X version: `sw_vers -productVersion`)

    GYP_DEFINES="mac_deployment_target=10.11" \
    make -j5 x64.release GYPFLAGS="-Dv8_use_external_startup_data=0 -Dv8_enable_i18n_support=0 -Dv8_enable_gdbjit=0"

On MacOS, the resulting libraries contain debugging information by default (even
though we've built the release version). As a result, the binaries are 30x
larger, then they should be. Strip that to reduce the size of the archives (and
build times!) very significantly:

    strip -S out/x64.release/libv8_*.a

Symlink the libraries and include directory to the Go package dir:

    ln -s `pwd`/out/x64.release ${GO_V8}/libv8
    ln -s `pwd`/include ${GO_V8}/include

## Reference

Also relevant is the v8 API release changes doc:

  https://docs.google.com/document/d/1g8JFi8T_oAE_7uAri7Njtig7fKaPDfotU6huOa1alds/edit


# Credits

This work is based off of several existing libraries:
  * https://github.com/fluxio/go-v8
  * https://github.com/kingland/go-v8
  * https://github.com/mattn/go-v8
