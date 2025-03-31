#!/bin/sh

export GOOS=${1:-"linux"}
export GOARCH=${2:-"amd64"}

go build -o build/${GOOS}-${GOARCH}/docker-deploy docker-deploy.go