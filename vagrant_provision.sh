#!/bin/bash
set -e

cd /vagrant

if [ ! -x /usr/bin/docker ] || [ ! -x /usr/bin/realpath ]; then
  echo "Installing Docker..."

  apt-key adv --keyserver hkp://pgp.mit.edu:80 --recv-keys 58118E89F3A912897C070ADBF76221572C52609D
  echo "deb https://apt.dockerproject.org/repo ubuntu-trusty main" > /etc/apt/sources.list.d/docker.list
  apt-get update -qy
  apt-get upgrade -qy
  apt-get install -y docker-engine realpath git
  adduser vagrant docker
  [ -x /vagrant/vagrant_provision_after_docker_installation.local ] && /vagrant/vagrant_provision_after_docker_installation.local

  echo "Installing Docker... Done."
fi

export GOPATH=/go
export PATH=$PATH:/usr/local/go/bin

if [ ! -d /go/src/github.com/cafebazaar ]; then
  mkdir -p /go/src/github.com/cafebazaar
  ln -s /vagrant /go/src/github.com/cafebazaar/blacksmith
  chown -R vagrant /go/
fi

if [ ! -f /usr/local/go/bin/go ]; then
  echo "Installing Go..."
  cd /tmp
  wget -cO go.tar.gz https://storage.googleapis.com/golang/go1.5.1.linux-amd64.tar.gz
  tar -C /usr/local -xzf go.tar.gz
  rm go.tar.gz

  echo 'export GOPATH=/go' >> ~/.profile
  echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.profile

  echo 'export GOPATH=/go' >> ~vagrant/.profile
  echo 'export PATH=$PATH:/usr/local/go/bin' >> ~vagrant/.profile

  echo "Installing Go... Done."
fi

cd /go/src/github.com/cafebazaar/blacksmith
sudo -u vagrant make docker
