#!/bin/bash

SCRIPT_DEBUG=true

DOCKER_IMAGE="${DOCKER_IMAGE:-cafebazaar/blacksmith}"
DOCKER_NAME=blacksmith_docker
DOCKER_EXEC="docker"

display_usage() {
  echo -e "Usage:"
  echo "$0 <workspace-dir> <etcd-endpoints> <interface> [other-args]"
  echo
  echo "Example:"
  echo "$0 ~/blacksmith-workspace http://example1.com:2379,http://example2.com:2379 etc0 -debug"
  echo
}

die() {
    echo "$@" 1>&2;
    echo
    exit 1
}

print_logs() {
	echo "Logs of the created container:"
	echo
	$DOCKER_EXEC logs $DOCKER_NAME
	echo
}

if [  $# -lt 3 ]; then
    display_usage
    exit 1
fi

WORKSPACE_DIR=$(realpath $1)
[ -d "$WORKSPACE_DIR" ] || die "Workspace does not exists: $WORKSPACE_DIR"
ETCD_ENDPOINTS=$2
INTERFACE=$3
OTHER_ARGS="${@:4}"

IP=$(ip addr | awk '/inet/ && /'$INTERFACE'/{sub(/\/.*$/,"",$2); print $2}')
[ "$IP" ] || die "Failed getting ip of the interface $INTERFACE"

# TODO: calculate instead of assuming the simple case
IFS=. read -r i1 i2 i3 i4 <<< "$IP"
[ "$i4" -lt 21 ] || echo "Warning: Your IP will be among those the dhcp is going to assign to the nodes! (Detected IP: $IP)" 1>&2

LEASE_START="$i1.$i2.$i3.31"
LEASE_RANGE=244
LEASE_SUBNET="255.255.255.0"
ROUTER="$i1.$i2.$i3.1"

DNS=$(grep nameserver /etc/resolv.conf | head -1 | awk '{print $2}')
[ "$DNS" ] || die "Failed getting DNS server from /etc/resolv.conf"

if [ SCRIPT_DEBUG ]; then
  echo
  echo "DOCKER_IMAGE:   $DOCKER_IMAGE"
  echo "WORKSPACE_DIR:  $WORKSPACE_DIR"
  echo "ETCD_ENDPOINTS: $ETCD_ENDPOINTS"
  echo "INTERFACE:      $INTERFACE"
  echo "OTHER_ARGS:     $OTHER_ARGS"
  echo
  echo "IP:             $IP"
	echo "ROUTER:         $ROUTER"
  echo "DNS:            $DNS"
	echo "LEASE_START:    $LEASE_START"
	echo "LEASE_RANGE:    $LEASE_RANGE"
	echo "LEASE_SUBNET:   $LEASE_SUBNET"
  echo
fi

VOLUME_ARGS="-v ${WORKSPACE_DIR}:/workspace"
ARGS="-etcd $ETCD_ENDPOINTS -if $INTERFACE -lease-start $LEASE_START -lease-range $LEASE_RANGE -lease-subnet $LEASE_SUBNET -router $ROUTER -dns $DNS -http-listen 0.0.0.0:8000 $OTHER_ARGS"

$DOCKER_EXEC kill -s HUP $DOCKER_NAME; $DOCKER_EXEC kill $DOCKER_NAME || echo "NOT FATAL"
$DOCKER_EXEC rm          $DOCKER_NAME || echo "NOT FATAL"
$DOCKER_EXEC run --name  $DOCKER_NAME --restart=always --net=host -d $VOLUME_ARGS $DOCKER_IMAGE $ARGS || die "Failed"

echo "Installed as $DOCKER_NAME, waiting for a few seconds..."
sleep 3

if [ "$(docker inspect $DOCKER_NAME | grep "\"Restarting\": false")" ]; then
	print_logs
  echo Seems OK.
  echo
else
	echo "Something went wrong for docker container $DOCKER_NAME"
	echo "Calculated args: $ARGS"
	print_logs
	echo "Killing the container..."
	$DOCKER_EXEC kill       $DOCKER_NAME && echo "Killed"
fi
