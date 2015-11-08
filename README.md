# AghaJoon, Bare-Metal Booting for CoreOS and Kubernetes

AghaJoon is a collection of DHCP, PXE, TFTP, and HTTP server, created with the
purpose of booting CoreOS on bare-metal machines and configuring them by using
cloud-config and ignition.

Booting a Linux system over the network is quite tedious. You have to
set up a TFTP server, configure your DHCP server to recognize PXE
clients, and send them the right set of magical options to get them to
boot, often fighting rubbish PXE ROM implementations.

AghaJoon aims to simplify this process, by packing the whole process
into a single binary that can cooperate with your network's existing
DHCP server.

## Running in Docker

AghaJoon is available as a Docker image called `cafebazaar/aghajoon`.

Because AghaJoon needs to listen for DHCP traffic, it has to run with
the host network stack.

```shell
sudo docker run -v workspace:/workspace --net=host cafebazaar/aghajoon -workspace /workspace
```

## Under the Hood
Check [this](docs/UnderTheHood.md).

## Development
You can use [Vagrant](https://www.vagrantup.com/) to quickly setup a test environment:

    (HOST)$ vagrant up --provider=libvirt pxeserver
    (HOST)$ vagrant ssh pxeserver
    (PXESERVER)$ wget http://alpha.release.core-os.net/amd64-usr/current/coreos_production_pxe.vmlinuz
    (PXESERVER)$ wget http://alpha.release.core-os.net/amd64-usr/current/coreos_production_pxe_image.cpio.gz
    (PXESERVER)$ aghajoon
    ### In another terminal
    (HOST)$ vagrant up --provider=libvirt pxeclient1

