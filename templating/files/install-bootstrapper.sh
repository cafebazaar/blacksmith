#!/bin/bash

## Installing Blacksmith Docker
  VOLUME_ARGS="-v /var/lib/blacksmith/workspaces:/workspace"
  ARGS="-etcd http://127.0.0.1:2379 -if $1 -cluster-name {{ (cluster_variable "cluster_name") }} -lease-start {{ (cluster_variable "internal_network_workers_start") }} -lease-range {{(cluster_variable "internal_network_workers_limit")}} -dns {{ (cluster_variable "external_dns") }} -file-server {{ (cluster_variable "file_server") }} -workspace-repo {{(cluster_variable "workspace-repo")}}"
  docker rm -f blacksmith || true
  docker pull {{ (cluster_variable "blacksmith_image") }}
  docker run --name blacksmith --restart=always -d --net=host $VOLUME_ARGS {{ (cluster_variable "blacksmith_image") }} $ARGS
