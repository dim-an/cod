#!/bin/bash

set -ex

name="cod-$(uname)"
mkdir -p "release/$name"
cp cod "release/$name/"
cd release

tar czf "$name.tgz" "$name"
