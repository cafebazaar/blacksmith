#!/bin/bash
set -e
source ./scripts/common.bash

BlacksmithContainer=blacksmith-kvm
EtcdContainer=blacksmith-kvm-etcd

if [ ! "$EUID" -ne 0 ]; then
  echo "Do not run as root"
  exit 1
fi

function run-etcd {
  docker run --name $EtcdContainer -d \
    -p 4001:4001 \
    -p 2380:2380 \
    -p 2379:2379 \
    quay.io/coreos/etcd:v2.2.3 \
    -name etcd0 \
    -advertise-client-urls http://${BobIP}:2379,http://${BobIP}:4001 \
    -listen-client-urls http://0.0.0.0:2379,http://0.0.0.0:4001 \
    -initial-advertise-peer-urls http://${BobIP}:2380 \
    -listen-peer-urls http://0.0.0.0:2380 \
    -initial-cluster-token etcd-cluster-1 \
    -initial-cluster etcd0=http://${BobIP}:2380 \
    -initial-cluster-state new
}

function run-blacksmith {
  mkdir -p ./config
  mkdir -p $GOPATH/src/github.com/cafebazaar/blacksmith/workspaces/current/
  if [ ! -d ${cert_dir} ]; then
    bash ./scripts/gencert.sh
  fi
  bash ./scripts/gen-blacksmith-config.bash > ./config/config.yaml
  bash ./scripts/gen-initial-yaml.bash > $GOPATH/src/github.com/cafebazaar/blacksmith/workspaces/current/initial.yaml
  bash ./scripts/download-coreos-images.bash $COREOS_CHANNEL $COREOS_VERSION

  docker run -d --name $BlacksmithContainer \
    --net=host \
    -v $GOPATH/src/github.com/cafebazaar/blacksmith/workspaces/current:/workspace \
    -v $PWD/config/:/config/ \
    -v /etc/ssl/certs:/etc/ssl/certs \
    ${BlacksmithImageForBob} --verbose --config /config/config.yaml
}

function usage {
  echo "USAGE:
  $0 all
  $0 destroy
  $0 create"
}

function main {
  sudo date
  case "$1" in
    "all")
      bash $0 destroy || true
      bash $0 create
      docker logs -f $BlacksmithContainer
      ;;
    "destroy")
      bash scripts/netctl.bash destroy
      docker rm -f $EtcdContainer 2>/dev/null || true
      docker rm -f $BlacksmithContainer 2>/dev/null || true
      destroy-nodes
      ;;
    "create")
      bash scripts/netctl.bash create
      run-etcd
      make blacksmith-agent && cp blacksmith-agent ./fs/blacksmith-agent
      run-blacksmith
      create-nodes
      ;;
    *)
      usage
      exit 2
      ;;
  esac
}

function create-nodes {
  local OPTS="--memory=1024 --vcpus=1 --pxe --disk pool=default,size=6 --os-type=linux --os-variant=generic --noautoconsole --events on_poweroff=preserve"
  virt-install --name $NODE1_NAME --network=bridge:$Bridge1,mac=$NODE1_MAC --network=bridge:$Bridge2 $OPTS --boot=hd,network
  virt-install --name $NODE2_NAME --network=bridge:$Bridge1,mac=$NODE2_MAC --network=bridge:$Bridge2 $OPTS --boot=hd,network
  virt-install --name $NODE3_NAME --network=bridge:$Bridge1,mac=$NODE3_MAC --network=bridge:$Bridge2 $OPTS --boot=hd,network
}

function destroy-nodes {
  for node in ${NODES[@]}; do
    virsh destroy $node
  done
  for node in ${NODES[@]}; do
    virsh undefine $node
  done
  virsh pool-refresh default
  for node in ${NODES[@]}; do
    virsh vol-delete --pool default $node.qcow2
  done
}

main $@
