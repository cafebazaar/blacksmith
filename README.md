# Blacksmith, Bare-Metal CoreOS Cluster Manager

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

## Test
For test you can use our ```blacksmith-kubernetes``` workspace.

### Cluster Setup
```bash
# Get the packages (ignore the warnings):
go get -v github.com/cafebazaar/blacksmith
go get -v github.com/cafebazaar/blacksmith-kubernetes

# Download the needed binaries of kubernetes workspace of blacksmith:
cd $GOPATH/src/github.com/cafebazaar/blacksmith-kubernetes/binaries
./download-all.sh

cd $GOPATH/src/github.com/cafebazaar/blacksmith-kubernetes
# Edit config.sh there and make it to suit your needs,
#
# If you happen to need a proxy config, you should edit the following lines:
# export CONTAINER_HTTP_PROXY=http://<your http proxy ip>:<port>
# export CONTAINER_HTTPS_PROXY=http://<your https proxy ip>:<port>
# Note: Referring localhost and 127.0.0.1 won't work here, good idea
# would be to make your local proxy server to listen on 0.0.0.0 and
# use your LAN IP here.
#
# You need to edit config.sh internal/external interface names to this:
# (or a real cluster scenario, according to your machines)
# export INTERNAL_INTERFACE_NAME=enp0s8
# export EXTERNAL_INTERFACE_NAME=enp0s9

# put your ssh keys into the cluster
echo "  - $(cat ~/.ssh/id_rsa.pub)" > ssh-keys.yaml

# Build workspace:
./build.sh

# Enter blacksmith
cd $GOPATH/src/github.com/cafebazaar/blacksmith
mkdir workspaces

# Initialize the cluster using VirtualBox
./dev_run.sh

# On blacksmith-kubernetes, once machines reached "installed" state, (click
# you can terminate BoB (local instance of blacksmith that has provisioned
# master machines with blacksmith-kubernetes workspace)

# Once download of blacksmith container and its requironments finished which
# takes minutes of even hours, for poor Internet connections
# (login with IP of third interface of boostrapper1 with `ssh core@IP`
# and enter `journalctl -f` to check that) you can add 5 workers to your
# just created virtual cluster. 
./dev_run.sh worker 5

# Append this line to your /etc/hosts
# <bootstrapper1 ip address>  test.cafecluster
#
kubectl --kubeconfig $GOPATH/src/github.com/cafebazaar/blacksmith-kubernetes/Takeaways/kubeconfig get nodes
```
