#!/usr/bin/env bash
set -e
/usr/bin/etcdctl --endpoints <<.EtcdCtlEndpoints>> watch /cafecluster/workspace-hash;
/usr/bin/curl -o /tmp/cloudconfig.yaml http://<<.WebServerAddr>>/t/cc/<<.Mac>>;
/usr/bin/coreos-cloudinit -validate -from-file /tmp/cloudconfig.yaml;
/usr/bin/coreos-install -d /dev/sda -c /tmp/cloudconfig.yaml -C beta -b <<.FileServerAddr>>;
/usr/bin/systemctl reboot
