# Blacksmith, Bare-Metal Booting for CoreOS and Kubernetes

[![GoDoc Widget]][GoDoc] [![Travis Widget]][Travis]

[GoDoc]: https://godoc.org/github.com/cafebazaar/blacksmith
[GoDoc Widget]: https://godoc.org/github.com/cafebazaar/blacksmith?status.png
[Travis]: https://travis-ci.org/cafebazaar/blacksmith
[Travis Widget]: https://travis-ci.org/cafebazaar/blacksmith.svg?branch=master

Blacksmith is a collection of DHCP, PXE, TFTP, and HTTP server,
created with the purpose of booting CoreOS on bare-metal machines and
configuring them by serving generated [cloud-config] and [ignition] files.

Warning: **UNDER HEAVY DEVELOPMENT**. The data-source model may dramatically
change in the near future. To be notified about the project getting more stable,
please subscribe to [this issue](https://github.com/cafebazaar/blacksmith/issues/5).

[cloud-config]: https://github.com/coreos/coreos-cloudinit
[ignition]: https://github.com/coreos/ignition

## Running in Docker

Blacksmith is available as a Docker image called `cafebazaar/blacksmith`.

Because Blacksmith needs to listen for DHCP traffic, it has to run with
the host network stack. You can use `install-as-docker.sh` to run
blacksmith as a docker container. The script has made some assumptions to
provide some of the required arguments of the `blacksmith` command.
To customize it according to your network layout, currently you have to edit
the script.

```shell
$ sudo ./install-as-docker.sh <workspace-path> <etcd-endpoints> <network-interface>
```

## Under the Hood
Check [this](docs/UnderTheHood.md).

## Development

*TODO: Add docker independent development instructions*

You can use [Vagrant](https://www.vagrantup.com/) to quickly setup a test environment:

```bash
(HOST)$ vagrant up --provider=libvirt pxeserver
(HOST)$ vagrant ssh pxeserver

### Clone and prepare workspace
(PXESERVER)$ cd ~
(PXESERVER)$ git clone https://github.com/cafebazaar/blacksmith-workspace-kubernetes.git
(PXESERVER)$ cd blacksmith-workspace-kubernetes
(PXESERVER)$ make update

### Run etcd as Docker service
(PXESERVER)$ sudo docker run -d -p 4001:4001 -p 2380:2380 -p 2379:2379 --restart=always --name etcd quay.io/coreos/etcd:v2.0.3  -name etcd0  -advertise-client-urls http://10.10.10.2:2379,http://10.10.10.2:4001  -listen-client-urls http://0.0.0.0:2379,http://0.0.0.0:4001  -initial-advertise-peer-urls http://10.10.10.2:2380  -listen-peer-urls http://0.0.0.0:2380  -initial-cluster-token etcd-cluster-1  -initial-cluster etcd0=http://10.10.10.2:2380  -initial-cluster-state new -cors '*'

### Install Blacksmith as Docker service
(PXESERVER)$ cd /vagrant
(PXESERVER)$ go generate
(PXESERVER)$ docker build -t cafebazaar/blacksmith
(PXESERVER)$ sudo ./install-as-docker.sh ~/blacksmith-workspace-kubernetes/workspace http://10.10.10.2:4001 eth1

### In another terminal
(HOST)$ vagrant up --provider=libvirt pxeclient1
```
