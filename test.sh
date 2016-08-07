#!/bin/sh -e

# generate dummy workspace
DUMMY_WORKSPACE=/tmp/blacksmith/workspaces/test-workspace
rm -rf $DUMMY_WORKSPACE
mkdir -p $DUMMY_WORKSPACE

echo "coreos-version: 1068.2.0" > $DUMMY_WORKSPACE/initial.yaml
echo "net-conf: '{\"netmask\": \"255.255.255.0\"}'" >> $DUMMY_WORKSPACE/initial.yaml

ETCD_RELEASE_VERSION=v2.3.7

export HostIP="127.0.0.1"

docker kill blacksmith-test-etcd || echo "wasn't running"
docker rm blacksmith-test-etcd || echo "didn't exist'"
docker run -d -p 127.0.0.1:20380:2380 -p 127.0.0.1:20379:2379 \
 --name blacksmith-test-etcd quay.io/coreos/etcd:$ETCD_RELEASE_VERSION \
 -name etcd0 \
 -advertise-client-urls http://${HostIP}:20379 \
 -listen-client-urls http://0.0.0.0:2379 \
 -initial-advertise-peer-urls http://${HostIP}:20380 \
 -listen-peer-urls http://0.0.0.0:2380 \
 -initial-cluster-token etcd-cluster-1 \
 -initial-cluster etcd0=http://${HostIP}:20380 \
 -initial-cluster-state new

# waiting for etcd to start
sleep 2

echo
echo "================================================================"
echo

# run all test
go get -t ./...
go build ./...
ETCD_ENDPOINT=http://${HostIP}:20379 go test -v ./...
