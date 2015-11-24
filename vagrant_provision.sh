#!/bin/bash
set -e

cd /vagrant

if [ ! -x /usr/bin/docker ]; then
  apt-key adv --keyserver hkp://pgp.mit.edu:80 --recv-keys 58118E89F3A912897C070ADBF76221572C52609D
  echo "deb https://apt.dockerproject.org/repo ubuntu-trusty main" > /etc/apt/sources.list.d/docker.list
  apt-get update -qy
  apt-get upgrade -qy
  apt-get install -y docker-engine
  [ -x /vagrant/vagrant_provision_after_docker_installation.local ] && /vagrant/vagrant_provision_after_docker_installation.local
fi

if [ ! -x /usr/bin/realpath ]; then
  apt-get update -qy
  apt-get upgrade -qy
  apt-get install -y realpath
fi

docker build -t cafebazaar/aghajoon .
