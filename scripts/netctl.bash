#/bin/env bash
set -e
source ./scripts/common.bash

create() {
  sudo ip link add name $Bridge type bridge
  sudo ip addr add $Subnet dev $Bridge
  sudo ip link set dev $Bridge up
}

destroy() {
  sudo ip link set dev $Bridge down
  sudo ip link del dev $Bridge
}

check() {
  ip addr show $Bridge
}

usage() {
  echo "USAGE: ${0##*/} <command>"
  echo "Commands:"
  echo "    create   create $Bridge bridge"
  echo "    destroy  destroy $Bridge bridge"
  echo "    check    check $Bridge bridge"
}

main() {
  case "$1" in
    "create") create;;
    "destroy") destroy;;
    "check") check;;
    *) usage; exit 2;;
  esac
}

main $@
