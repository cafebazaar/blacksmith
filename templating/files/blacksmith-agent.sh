#!/bin/bash

/usr/bin/mkdir -p /opt/bin/
/usr/bin/wget -O /opt/bin/agent {{ blacksmith_variable "agent-url" }}
/usr/bin/chmod +x /opt/bin/agent

/opt/bin/agent \
    -debug \
    -mac {{ (machine_variable "mac") }} \
    -etcd http://{{ if (machine_variable "state") | eq "unknown" }}{{ cluster_variable "bob" }}{{ else }}{{ cluster_variable "master" }}{{ end }}:2379 \
    -cluster-name {{ blacksmith_variable "cluster-name" }} \
    -heartbeat-server {{ if (machine_variable "state") | eq "unknown" }}{{ cluster_variable "bob" }}{{ else }}{{ cluster_variable "master" }}{{ end }}:8001 \
    -cloudconfig-url  http://{{ if (machine_variable "state") | eq "unknown" }}{{ cluster_variable "bob" }}{{ else }}{{ cluster_variable "master" }}{{ end }}:8000/t/cc/{{ machine_variable "mac" }} \
    -file-server {{ blacksmith_variable "file-server" }} \
    -tls-cert {{ blacksmith_variable "agent-tls-cert" }} \
    -tls-key {{ blacksmith_variable "agent-tls-key" }} \
    -tls-ca {{ blacksmith_variable "agent-tls-ca" }}
