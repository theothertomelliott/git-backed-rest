#!/bin/bash

PACKAGE_DIR=$(dirname "$0")
TESTDATA_DIR=$PACKAGE_DIR/testdata

echo $TESTDATA_DIR

go test -trace=$TESTDATA_DIR/trace.out ./$PACKAGE_DIR
go tool trace $TESTDATA_DIR/trace.out