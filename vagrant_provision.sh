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
  wget -cO go.tar.gz https://storage.googleapis.com/golang/go1.5.1.linux-amd64.tar.gz || wget -cO go.tar.gz http://netix.dl.sourceforge.net/project/gnuhub/go1.5.1.linux-amd64.tar.gz
  tar -C /usr/local -xzf go.tar.gz
  rm go.tar.gz

  echo 'export GOPATH=/go' >> ~/.profile
  echo 'export PATH=$PATH:/usr/local/go/bin:/go/bin' >> ~/.profile

  echo 'export GOPATH=/go' >> ~vagrant/.profile
  echo 'export PATH=$PATH:/usr/local/go/bin:/go/bin' >> ~vagrant/.profile

  echo "Installing Go... Done."
fi

echo
echo "Total number of pxeservers: $1"
echo "Index number of this pxeserver: $2"
echo "Workspace: $3"
echo

[ -d "$3" ] || echo "Workspace doesn't exists. Please prepare it and reprovision this instance."
[ -d "$3" ] || exit 1

ETCD_CLUSTER=""
ETCD_ENDPOINTS=""
for i in `seq 1 $1`; do
  if [ "$i" -gt "1" ]; then
    ETCD_CLUSTER="${ETCD_CLUSTER},"
    ETCD_ENDPOINTS="${ETCD_ENDPOINTS},"
  fi
  ETCD_CLUSTER="${ETCD_CLUSTER}etcd${i}=http://10.10.10.1${i}:2380"
  ETCD_ENDPOINTS="${ETCD_ENDPOINTS}http://10.10.10.1${i}:2379"
done

[ "$4" -eq "1" ] && docker kill etcd || echo "OK!"
[ "$4" -eq "1" ] && docker rm etcd || echo "OK!"
docker run -d -p 2379:2379 -p 2380:2380 --restart=always --name etcd quay.io/coreos/etcd:v2.2.4 \
 -name etcd${2}   -advertise-client-urls http://10.10.10.1${2}:2379 \
 -listen-client-urls http://0.0.0.0:2379 \
 -initial-advertise-peer-urls http://10.10.10.1${2}:2380 \
 -listen-peer-urls http://0.0.0.0:2380 \
 -initial-cluster-token etcd-cluster \
 -initial-cluster $ETCD_CLUSTER \
 -initial-cluster-state new -cors '*' || [ "$4" -ne "1" ]

[ "$4" -eq "1" ] && docker kill skydns || echo "OK!"
[ "$4" -eq "1" ] && docker rm skydns || echo "OK!"
docker inspect skydns || docker run -d -p 53:53 --restart=always --name skydns -e ETCD_MACHINES=$ETCD_ENDPOINTS skynetservices/skydns
sudo -u vagrant /vagrant/vagrant_make.sh || [ "$4" -ne "1" ]

exec /vagrant/install-as-docker.sh $3 $ETCD_ENDPOINTS eth1
