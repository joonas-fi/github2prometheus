#!/bin/bash -eu

source /build-common.sh

BINARY_NAME="github2prometheus"
COMPILE_IN_DIRECTORY="cmd/github2prometheus"

standardBuildProcess

buildstep packageLambdaFunction
