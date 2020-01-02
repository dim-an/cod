.PHONY: all build test release

THISDIR := $(realpath $(dir $(firstword $(MAKEFILE_LIST))))
GIT_SHA := $(shell git rev-parse HEAD)

all: build

build:
	cd ${THISDIR}
	go build -o cod -ldflags "-X main.GitSha=`git rev-parse HEAD`"

test: build
	cd ${THISDIR}
	env COD_TEST_BINARY="${THISDIR}/cod" go test ./...

release: build
	cd ${THISDIR}
	python release.py
