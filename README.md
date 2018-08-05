# V8 Bindings for Go [![Build Status](https://travis-ci.org/augustoroman/v8.svg?branch=master)](https://travis-ci.org/augustoroman/v8) [![Go Report Card](https://goreportcard.com/badge/github.com/augustoroman/v8)](https://goreportcard.com/report/github.com/augustoroman/v8) [![GoDoc](https://godoc.org/github.com/augustoroman/v8?status.svg)](https://godoc.org/github.com/augustoroman/v8)

The v8 bindings allow a user to execute javascript from within a go executable.

The bindings are tested to work with several recent v8 builds matching the
Chrome builds 54 - 60 (see the .travis.yml file for specific versions). For
example, Chrome 59 (dev branch) uses v8 5.9.211.4 when this was written.

Note that v8 releases match the Chrome release timeline:
Chrome 48 corresponds to v8 4.8.\*, Chrome 49 matches v8 4.9.\*. You can see
the table of current chrome and the associated v8 releases at:

http://omahaproxy.appspot.com/

# Using a pre-compiled v8

v8 is very slow to compile, it's a large project. If you want to go that route, there are building instructions below.

Fortunately, there's a project that pre-builds v8 for various platforms. It's packaged as a ruby gem called [libv8](https://rubygems.org/gems/libv8).

```bash
# Find the appropriate gem version for your OS,
# visit: https://rubygems.org/gems/libv8/versions

# Download the gem
# MacOS Sierra is darwin-16, for v8 6.3.292.48.1 it looks like:
curl https://rubygems.org/downloads/libv8-6.3.292.48.1-x86_64-darwin-16.gem > libv8.gem

# Extract the gem (it's a tarball)
tar -xf libv8.gem

# Extract the `data.tar.gz` within
cd libv8-6.3.292.48.1-x86_64-darwin-16
tar -xzf data.tar.gz

# Symlink the compiled libraries and includes
ln -s $(pwd)/data/vendor/v8/include $GOPATH/src/github.com/augustoroman/v8/include
ln -s $(pwd)/data/vendor/v8/out/x64.release $GOPATH/src/github.com/augustoroman/v8/libv8

# Run the tests to make sure everything works
cd $GOPATH/src/github.com/augustoroman/v8
go test
```

# Using docker (linux only)

For linux builds, you can use pre-built libraries or build your own.

## Pre-built versions

To use a pre-built library, select the desired v8 version from https://hub.docker.com/r/augustoroman/v8-lib/tags/ and then run:

```bash
# Select the v8 version to use:
export V8_VERSION=6.7.77
docker pull augustoroman/v8-lib:$V8_VERSION           # Download the image, updating if necessary.
docker rm v8 ||:                                      # Cleanup from before if necessary.
docker run --name v8 augustoroman/v8-lib:$V8_VERSION  # Run the image to provide access to the files.
docker cp v8:/v8/include include/                     # Copy the include files.
docker cp v8:/v8/lib libv8/                           # Copy the library fiels.
```

## Build your own via docker

This takes a lot longer, but is still easy:

```bash
export V8_VERSION=6.7.77
docker build --build-arg V8_VERSION=$V8_VERSION --tag augustoroman/v8-lib:$V8_VERSION docker-v8-lib/
```

and then extract the files as above:

```bash
docker rm v8 ||:                                      # Cleanup from before if necessary.
docker run --name v8 augustoroman/v8-lib:$V8_VERSION  # Run the image to provide access to the files.
docker cp v8:/v8/include include/                     # Copy the include files.
docker cp v8:/v8/lib libv8/                           # Copy the library fiels.
```

# Building v8

## Prep

You need to build v8 statically and place it in a location cgo knows about. This requires special tooling and a build directory. Using the [official instructions](https://github.com/v8/v8/wiki/Building-from-Source) as a guide, the general steps of this process are:

1.  `go get` the binding library (this library)
1.  Create a v8 build directory
1.  [Install depot tools](http://commondatastorage.googleapis.com/chrome-infra-docs/flat/depot_tools/docs/html/depot_tools_tutorial.html#_setting_up)
1.  Configure environment
1.  Download v8
1.  Build v8
1.  Copy or symlink files to the go library path
1.  Build the bindings

```
go get github.com/augustoroman/v8
export V8_GO=$GOPATH/src/github.com/augustoroman/v8
export V8_BUILD=$V8_GO/v8/build #or wherever you like
mkdir -p $V8_BUILD
cd $V8_BUILD
git clone https://chromium.googlesource.com/chromium/tools/depot_tools.git
export PATH=$PATH:$V8_BUILD/depot_tools
fetch v8 #pull down v8 (this will take some time)
cd v8
git checkout 6.7.77
gclient sync
```

## Linux

```
./build/install-build-deps.sh #only needed once
gn gen out.gn/golib --args="strip_debug_info=true v8_use_external_startup_data=false v8_enable_i18n_support=false v8_enable_gdbjit=false v8_static_library=true symbol_level=0 v8_experimental_extra_library_files=[] v8_extra_library_files=[]"
ninja -C out.gn/golib
# go get some coffee
```

## OSX

```
gn gen out.gn/golib --args="is_official_build=true strip_debug_info=true v8_use_external_startup_data=false v8_enable_i18n_support=false v8_enable_gdbjit=false v8_static_library=true symbol_level=0 v8_experimental_extra_library_files=[] v8_extra_library_files=[]"
ninja -C out.gn/golib
# go get some coffee
```

## Symlinking

Now you can create symlinks so that cgo can associate the v8 binaries with the go library.

```
cd $V8_GO
./symlink.sh $V8_BUILD/v8
```

## Verifying

You should be done! Try running `go test`

# Reference

Also relevant is the v8 API release changes doc:

https://docs.google.com/document/d/1g8JFi8T_oAE_7uAri7Njtig7fKaPDfotU6huOa1alds/edit

# Credits

This work is based off of several existing libraries:

* https://github.com/fluxio/go-v8
* https://github.com/kingland/go-v8
* https://github.com/mattn/go-v8
