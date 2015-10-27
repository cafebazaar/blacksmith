# AghaJoon, Bare-Metal Kubernetes Cluster Manager

```
There once was a protocol called PXE,
Whose specification was overly tricksy.
A committee refined it
Into a big Turing tarpit,
And now you're using it to boot your PC.
```

Booting a Linux system over the network is quite tedious. You have to
set up a TFTP server, configure your DHCP server to recognize PXE
clients, and send them the right set of magical options to get them to
boot, often fighting rubbish PXE ROM implementations.

Pixiecore aims to simplify this process, by packing the whole process
into a single binary that can cooperate with your network's existing
DHCP server.

### CoreOS

Pixiecore was originally written as a component in an automated
installation system for CoreOS on bare metal. For this example, let's
set up a netboot for the alpha CoreOS release:

```shell
# Grab the PXE images and verify them
wget http://alpha.release.core-os.net/amd64-usr/current/coreos_production_pxe.vmlinuz
wget http://alpha.release.core-os.net/amd64-usr/current/coreos_production_pxe_image.cpio.gz

# In the real world, you would AUTHENTICATE YOUR DOWNLOADS
# here. CoreOS distributes image signatures, but that only really
# helps if you already know the right GPG key.

# Go!
pixiecore -kernel coreos_production_pxe.vmlinuz -initrd coreos_production_pxe_image.cpio.gz --cmdline coreos.autologin
```

## Running in Docker

Pixiecore is available as a Docker image called
`cafebazaar/aghajoon`. It's an automatic Docker Hub build that tracks
the repository.

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
    (PXESERVER)$ pixiecore -debug -kernel coreos_production_pxe.vmlinuz -initrd coreos_production_pxe_image.cpio.gz --cmdline coreos.autologin
    ### In another terminal
    (HOST)$ vagrant up --provider=libvirt pxeclient1

