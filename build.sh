#!/bin/sh

go-bindata -pkg="httpgen_generator" -o="httpgen_generator/bindata.go" templates
go build
pushd webui;gopherjs build; popd