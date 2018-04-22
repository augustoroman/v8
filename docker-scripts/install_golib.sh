#!/bin/bash -ex

: "${CHROMIUM_DIR:?CHROMIUM_DIR must be set}"

cd $(go env GOPATH)/src/github.com/augustoroman/v8
./symlink.sh ${CHROMIUM_DIR}/v8
go install .
go test .
