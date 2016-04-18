# Blacksmith, Bare-Metal CoreOS Cluster Manager

[![GoDoc Widget]][GoDoc] [![Travis Widget]][Travis]

[GoDoc]: https://godoc.org/github.com/cafebazaar/blacksmith
[GoDoc Widget]: https://godoc.org/github.com/cafebazaar/blacksmith?status.png
[Travis]: https://travis-ci.org/cafebazaar/blacksmith
[Travis Widget]: https://travis-ci.org/cafebazaar/blacksmith.svg?branch=master

Blacksmith is a collection of DHCP, PXE, TFTP, and HTTP servers,
created with the purpose of booting CoreOS on bare-metal machines,
configuring them by serving generated [cloud-config] and [ignition] files, and
maintaining the cluster over the time.
Blacksmith uses etcd to store the states, and to elect a leader. So you can run
multiple instances of Blacksmith to provide a high available CoreOS over bare-metal
infrastructure.

Warning: **UNDER DEVELOPMENT**. To be notified about the project getting more stable,
please subscribe to [this issue](https://github.com/cafebazaar/blacksmith/issues/5).

![Screenshot of Nodes List page - Blacksmith][screenshot]

[cloud-config]: https://github.com/coreos/coreos-cloudinit
[ignition]: https://github.com/coreos/ignition
[screenshot]: https://github.com/cafebazaar/blacksmith/raw/master/docs/NodesList.png "Nodes List - Blacksmith"

## Workspace and Templating

The cloud-config and ignition files, and the bootparams string which is passed
to the kernel at boot time, are provided by executing templates for each machine.
These templates, along with CoreOS images and other binary files forms the
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
Some softwares (Kubernetes?) count on it. To provide similar functionality, you
need to run [SkyDNS] on the same instances you run Blacksmith on. Blacksmith will
configure them through etcd.

[SkyDNS]: https://github.com/skynetservices/skydns

## Documentation
Check [this](docs/README.md).

## Development

You can use [Vagrant](https://www.vagrantup.com/) to quickly setup a test environment:

```bash
make blacksmith

### Clone and prepare a workspace
mkdir workspaces
cd workspaces
git clone https://github.com/cafebazaar/blacksmith-kubernetes.git
cd blacksmith-kubernetes/binaries
./download-all.sh
cd ..
# put your key into ssh-keys.yaml
./build.sh
cd ..
ln -s blacksmith-kubernetes/workspace current

# Start 3 machines, which will be provisioned to serve a 3-node etcd cluster,
# 3 working instances of SkyDNS, and a 3-node Blacksmith cluster
vagrant up --provider=libvirt

### Check the logs
vagrant ssh pxeserver1 -c "docker logs -f blacksmith_docker"

### In another terminal, start a client machine
vagrant up --provider=libvirt pxeclient1
```
