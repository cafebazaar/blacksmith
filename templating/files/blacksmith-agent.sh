#!/bin/bash

/usr/bin/mkdir -p /opt/bin/
/usr/bin/wget -O /opt/bin/agent {{ blacksmith_variable "agent-url" }}
/usr/bin/chmod +x /opt/bin/agent

/opt/bin/agent \
    --debug \
    --mac {{ (machine_variable "mac") }} \
    --etcd http://127.0.0.1:2379 \
    --cluster-name {{ (blacksmith_variable "cluster-name") }} \
    --heartbeat-server 127.0.0.1:8001 \
    --cloudconfig-url http://127.0.0.1:8000/t/cc/{{ (machine_variable "mac") }} \
    --tls-cert {{ blacksmith_variable "agent-tls-cert" }} \
    --tls-key {{ blacksmith_variable "agent-tls-key" }} \
    --tls-ca {{ blacksmith_variable "tls-ca" }}
