#!/bin/bash

/usr/bin/mkdir -p /opt/bin/
/usr/bin/wget -O /opt/bin/agent {{ blacksmith_variable "agent-url" }}
/usr/bin/chmod +x /opt/bin/agent

# {{if (machine_variable "state") | eq "unknown"}}BOB{{else}}master.cafecluster{{end}}

/opt/bin/agent \
    -debug \
    -mac {{ (machine_variable "mac") }} \
    -etcd http://{{ if (machine_variable "state") | eq "unknown" }}{{ cluster_variable "bob" }}{{ else }}master.cafecluster{{ end }}:2379 \
    -cluster-name {{ blacksmith_variable "cluster-name" }} \
    -heartbeat-server {{ if (machine_variable "state") | eq "unknown" }}{{ cluster_variable "bob" }}{{ else }}master.cafecluster{{ end }}:8001 \
    -cloudconfig-url  http://{{ if (machine_variable "state") | eq "unknown" }}{{ cluster_variable "bob" }}{{ else }}master.cafecluster{{ end }}:8000/t/cc/{{ machine_variable "mac" }} \
    -tls-cert {{ blacksmith_variable "agent-tls-cert" }} \
    -tls-key {{ blacksmith_variable "agent-tls-key" }} \
    -tls-ca {{ blacksmith_variable "agent-tls-ca" }}
