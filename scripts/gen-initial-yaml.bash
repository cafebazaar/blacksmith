#!/bin/bash
source ./scripts/common.bash

user=$(whoami)
pubkey=$(cat ~/.ssh/id_rsa.pub)

cat <<YAML
cluster-variables:
  coreos-version: "${COREOS_VERSION}"
  net-conf: '{"netmask":"255.255.255.0", "classlessRouteOption": [{"router": "172.20.0.1", "size":24, "destination": "172.20.0.0"}]}'
  cluster_name: "cafecluster"
  internal_net_conf: '{"netmask":"255.255.255.0", "classlessRouteOption": [{"router": "172.20.0.1", "size":24, "destination": "172.20.0.0"}]}'
  bootstrapper1_hostname: "bootstrapper1"
  bootstrapper1_ip: "172.20.0.11"
  bootstrapper2_hostname: "bootstrapper2"
  bootstrapper2_ip: "172.20.0.12"
  bootstrapper3_hostname: "bootstrapper3"
  bootstrapper3_ip: "172.20.0.13"
  internal_network_netsize: "24"
  external_network_netsize: "24"
  internal_network_gateway_ip: "172.20.0.1"
  external_dns: "8.8.8.8"
  pod_network: "10.1.0.0/16"
  service_ip_range: "10.100.0.0/16"
  k8s_service_ip: "10.100.0.1"
  dns_server_ip: "10.100.0.10"
  k8s_lb_dns: "k8s.roo.cloud"
  gateway: "172.21.0.1"
  http_proxy: "172.20.0.1:8118"
  https_proxy: "172.20.0.1:8118"
  k8s_version: "v1.4.5_coreos.0"
  bob: $BobIP
  master: master.cafecluster
  hyperkube_image: "quay.io/coreos/hyperkube:v1.6.1_coreos.0"
  ca: "$(cat ./certs/ca.crt | base64 -w0)"
  apiserver-cert: "$(cat ./certs/server.crt | base64 -w0)"
  apiserver-key: "$(cat ./certs/server.key | base64 -w0)"
  machine-cert: "$(cat ./certs/client.crt | base64 -w0)"
  machine-key: "$(cat ./certs/client.key | base64 -w0)"
machines:
  "${NODE1_MAC}":
    hostname: "bootstrapper1"
    internal_interface_name: "ens3"
    external_interface_name: "ens4"
    external_ip: "172.21.0.11"
    blacksmith_server: "true"
    state: "unknown"
    _machine: '{"ip":"172.20.0.11","first_seen":0,"type":1}'
    mac: "$NODE1_MAC"
  "${NODE2_MAC}":
    hostname: "bootstrapper2"
    internal_interface_name: "ens3"
    external_interface_name: "ens4"
    external_ip: "172.21.0.12"
    blacksmith_server: "true"
    state: "unknown"
    _machine: '{"ip":"172.20.0.12","first_seen":0,"type":1}'
    mac: "$NODE2_MAC"
  "${NODE3_MAC}":
    hostname: "bootstrapper3"
    internal_interface_name: "ens3"
    external_interface_name: "ens4"
    external_ip: "172.21.0.13"
    blacksmith_server: "true"
    state: "unknown"
    _machine: '{"ip":"172.20.0.13","first_seen":0,"type":1}'
    mac: "$NODE3_MAC"
ssh-keys:
  $user: "$pubkey"
YAML
