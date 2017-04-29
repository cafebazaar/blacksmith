# Blacksmith, Bare-Metal CoreOS Cluster Manager

[![Go Report Card](https://goreportcard.com/badge/github.com/cafebazaar/blacksmith)](https://goreportcard.com/report/github.com/cafebazaar/blacksmith)
[![Travis widget]][Travis] [![wercker widget]][wercker] [![Quay widget]][Quay]
![Status](https://img.shields.io/badge/status-under%20development-orange.svg)

[Travis]: https://travis-ci.org/cafebazaar/blacksmith "Continuous Integration"
[Travis widget]: https://travis-ci.org/cafebazaar/blacksmith.svg?branch=master
[wercker]: https://app.wercker.com/project/bykey/3f1066d1d6886dfc62a9469da691c1c3 "Container Build System"
[wercker widget]: https://app.wercker.com/status/3f1066d1d6886dfc62a9469da691c1c3/s/master
[Quay]: https://quay.io/repository/cafebazaar/blacksmith "Docker Repository on Quay"
[Quay widget]: https://quay.io/repository/cafebazaar/blacksmith/status

Blacksmith is a collection of DHCP, PXE, TFTP, and HTTP servers,
created with the purpose of booting CoreOS on bare-metal machines,
configuring them by serving generated [cloud-config] and [ignition] files, and
maintaining the cluster over time.
Blacksmith uses [etcd](https://coreos.com/etcd/) to store the states, and to elect a leader. So you can run
multiple instances of Blacksmith to provide a high available CoreOS over bare-metal
infrastructure.

**Warning:** This project is under development. To be notified about the project becoming more stable,
please subscribe to [this issue](https://github.com/cafebazaar/blacksmith/issues/5).

![Screenshot of Nodes List page - Blacksmith][screenshot]

[cloud-config]: https://github.com/coreos/coreos-cloudinit
[ignition]: https://github.com/coreos/ignition
[screenshot]: https://github.com/cafebazaar/blacksmith/raw/master/docs/NodesList.png "Nodes List - Blacksmith"

## Workspace and Templating

The cloud-config and ignition files, and the bootparams string which is passed
to the kernel at boot time, are provided by executing templates for each machine.
These templates, along with CoreOS images and other binary files, form the
runtime environment of your cluster. In Blacksmith, we call this folder *Workspace*.
For more information on the structure of a workspace, check the [workspace doc].

[workspace doc]: docs/Workspace.md

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

## DNS
In some IaaS environments, machine names are resolvable in the internal network.
Some software (Kubernetes?) count on it. To provide similar functionality, you
need to run [SkyDNS] on the same instances you run Blacksmith on. Blacksmith will
configure them through etcd.

[SkyDNS]: https://github.com/skynetservices/skydns

## Documentation
Check [this](docs/README.md).

## Development

Get packages:
```bash
go get github.com/cafebazaar/blacksmith
cd $GOPATH/github.com/cafebazaar/blacksmith
```

Run tests:
```bash
make run_test_etcd test
```

Download CoreOS images and create a cluster of 3 KVM virtual machines:
```bash
bash scripts/kvm-run.bash all
```

Run a file server to serve files downloaded in the `fs/` directory.

After CoreOS is booted, install it on disk:

```bash
bash scripts/install-nodes.bash
```

Reboot the nodes after CoreOS is installed on disk

```bash
bash scripts/reboot-nodes.bash
```

## Kubernetes

Add this line to your ```/etc/hosts``` file:
```
<bootstrapper1 ip address>  test.cafecluster
```
Kubernetes setup takes several minutes. For poor internet connection it may takes several hours. After that you can use kubernetes via ```kubectl``` command as follows:
```bash
kubectl --kubeconfig $GOPATH/src/github.com/cafebazaar/blacksmith-kubernetes/Takeaways/kubeconfig get nodes
```
