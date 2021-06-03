.RECIPEPREFIX := >
.PHONY: all build test install

THISDIR := $(realpath $(dir $(firstword $(MAKEFILE_LIST))))
GIT_SHA := $(shell git rev-parse HEAD)

GO_FLAGS := -ldflags "-X main.GitSha=${GIT_SHA}"

ifdef OUTPUTDIR
	REAL_OUTPUTDIR := $(realpath ${OUTPUTDIR})
	ifndef REAL_OUTPUTDIR
		$(error OUTPUTDIR=${OUTPUTDIR} directory does not exist)
	endif
	OUTPUTDIR := ${REAL_OUTPUTDIR}
else
	OUTPUTDIR := ${THISDIR}
endif

all: build

build:
> cd ${THISDIR}
> go build -o "${OUTPUTDIR}/cod" ${GO_FLAGS}

test: build
> cd ${THISDIR}
> env COD_TEST_BINARY="${OUTPUTDIR}/cod" go test ./...

install: build
> cd ${THISDIR}
> python release.py
