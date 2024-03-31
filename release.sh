#!/bin/bash
tag=`git describe --tags`
commit=`git rev-parse --short HEAD`
VERSION="${tag}-${commit}" goreleaser --clean
