#!/usr/bin/env bash

echo -n "WITHOUT TESTS "
find . -name '*.go' -and -not -regex '.*test.*' -exec cat '{}' + | wc -l

echo -n "TESTS ONLY    "
find . -name '*.go' -and -regex '.*test.*' -exec cat '{}' + | wc -l

echo -n "ALL SOURCES   "
find . -name '*.go' -exec cat '{}' + | wc -l
