#!/bin/sh

# generate dummy workspace
DUMMY_WORKSPACE=/tmp/blacksmith/workspaces/test-workspace
mkdir -p $DUMMY_WORKSPACE

echo "coreos-version: 1068.2.0" > $DUMMY_WORKSPACE/initial.yaml


# install dependencies
go get github.com/coreos/etcd

# etcd configs flags
ETCD_NAME=blacksmith_testing
ETCD_DATA_DIR=/tmp/blacksmith/etcd/$ETCD_NAME.etcd
ETCD_CORS=*

# clean etcd generated data
rm -rf $ETCD_DATA_DIR

# build etcd
go build -o /tmp/gopath/src/github.com/coreos/etcd/etcd.run github.com/coreos/etcd

# run etcd instance and save pid of it's proccess
/tmp/gopath/src/github.com/coreos/etcd/etcd.run & export ETCD_PID=$!

# wtf sleep
sleep 2

# run all test
cd datasource
go test

# kill etcd proccess
kill -9 $ETCD_PID 