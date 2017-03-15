#!/bin/bash

/usr/bin/mkdir -p /opt/bin/
/usr/bin/wget -O /opt/bin/agent {{ blacksmith_variable "agent-url" }}
/usr/bin/chmod +x /opt/bin/agent

/opt/bin/agent \
    --debug \
    --mac {{ (machine_variable "mac") }} \
    --etcd http://127.0.0.1:2379 \
    --cluster-name {{ blacksmith_variable "cluster-name" }} \
    --heartbeat-server {{ machine_variable "external_ip" }}:8001 \
    --cloudconfig-url {{ machine_variable "external_ip" }}:8000/t/cc/{{ machine_variable "mac" }} \
    --agent-tls-cert {{ blacksmith_variable "agent-tls-cert" }} \
    --agent-tls-key {{ blacksmith_variable "agent-tls-key" }} \
    --agent-tls-ca {{ blacksmith_variable "agent-tls-ca" }}
