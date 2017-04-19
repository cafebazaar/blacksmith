#/bin/env bash
set -x
source ./scripts/common.bash

create() {
  local br=$1
  local subnet=$2
  sudo ip link add name $br type bridge
  sudo ip addr add $subnet dev $br
  sudo ip link set dev $br up
  sudo sysctl -w net.ipv4.ip_forward=1
  sudo iptables --table nat --append POSTROUTING --out-interface $InternetInterface -j MASQUERADE
  sudo iptables --insert FORWARD --in-interface $br -j ACCEPT
  # To test, you could setup default gateway on a VM:
  # $ sudo ip route add default via 172.20.0.1
}

destroy() {
  local br=$1
  sudo ip link set dev $br down
  sudo ip link del dev $br
}

check() {
  local br=$1
  ip addr show $br
}

usage() {
  echo "USAGE: ${0##*/} <command>"
  echo "Commands:"
  echo "    create   create $Bridge1 and $Bridge2 bridge"
  echo "    destroy  destroy $Bridge1 and $Bridge2 bridge"
  echo "    check    check $Bridge1 and $Bridge2  bridge"
}

main() {
  case "$1" in
    "create")
      create $Bridge1 $Subnet1
      create $Bridge2 $Subnet2
      ;;
    "destroy")
      destroy $Bridge1
      destroy $Bridge2
      ;;
    "check")
      check $Bridge1
      check $Bridge2
      ;;
    *) usage; exit 2;;
  esac
}

main $@
