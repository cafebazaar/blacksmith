#!/bin/bash
set -o errexit
set -o pipefail
set -o nounset
# set -o xtrace

if ! which vboxmanage > /dev/null; then
    echo 'You should install VirtualBox or make sure "vboxmanage" is in path'
    exit 1
fi

if ! which virtualbox > /dev/null; then
    echo 'You should install "virtualbox-qt" also'
    exit 1
fi

if [[ ! $(vboxmanage list extpacks | grep "Oracle VM VirtualBox Extension Pack") ]]; then
    echo 'You should make sure if "virtualbox-ext-pack" is installed which is needed for PXE boot thing, its installation however could be tricky (needs proxy)'
    exit 1
fi

if ! which docker > /dev/null; then
    echo 'You should install Docker, needed for local etcd'
    exit 1
fi


####
# BoB IP
HostIP="172.19.1.1"
# hostonly network name, it is "vboxnet0" by default and we have less control for what it should be it seems
HOSTONLY="vboxnet0"
if [[ -f ".vbox_network_hostonly_if" ]]; then HOSTONLY=$(cat .vbox_network_hostonly_if); fi
# detects which interface connects us to the Internet, needed for bridge
INTERTNETIF=$(route | grep '^default' | grep -o '[^ ]*$' || true)
if [[ -z "$INTERTNETIF" ]]; then
    echo "Default route not found"
    exit 1
fi

# number of bootstrapper, 3 is good enough usually
BOOTSTRAPPERS=3
# number of workers
WORKERS=3
if [[ -f ".vbox_number_of_workers" ]]; then WORKERS=$(cat .vbox_number_of_workers); fi
# VirtualBox's default VM folder, usually is "/home/$USER/VirtualBox VMs"
VMDIR=$(vboxmanage list systemproperties | grep 'Default machine folder' | grep -o '/.*')


####
function createNetwork {
    # usually it installs a hostonly network named "vboxnet0", it would be nice if we could control its name, it seems we can't, however
    HOSTONLY=$(vboxmanage hostonlyif create 2>/dev/null | sed "s/.*'\(.*\)'.*/\1/g")
    echo $HOSTONLY > .vbox_network_hostonly_if
    vboxmanage hostonlyif ipconfig $HOSTONLY --ip 172.19.1.1 --netmask 255.255.255.0
}

function create_machine {

    local MAC3=$2
    local MAC4=$3

    vboxmanage createvm --name $1 --ostype "Linux_64"

    vboxmanage registervm "$VMDIR/$1/$1.vbox"

    vboxmanage modifyvm $1 \
        --memory 4096 \
        --nic1 hostonly \
        --nictype1 82540EM \
        --hostonlyadapter1 $HOSTONLY \
        --nicpromisc1 allow-all \
        --nic2 bridged \
        --nictype2 82540EM \
        --bridgeadapter2 $INTERTNETIF \
        --nicpromisc2 allow-all \
        --boot1 disk \
        --boot2 net \
        --boot3 none \
        --boot4 none \
        --macaddress1 $MAC3 \
        --macaddress2 $MAC4
    

    vboxmanage storagectl $1 \
        --name IDE0 \
        --add sata \
        --bootable on \
        2>/dev/null

    vboxmanage createhd \
        --filename "$VMDIR/$1/$1" \
        --size 20000

    vboxmanage storageattach $1 \
        --storagectl IDE0 \
        --port 0 \
        --device 0 \
        --type hdd \
        --medium "$VMDIR/$1/$1.vdi"
}

function createBootstrappers {
    for i in $(seq $BOOTSTRAPPERS); do
        MAC="00027d15be8$((i*2))"
        MAC1="00027d13be8$((i*2))"
        create_machine bootstrapper_$i $MAC $MAC1
    done
}

function removeMachines {
    for i in $(seq $BOOTSTRAPPERS); do
        vboxmanage controlvm bootstrapper_$i poweroff 2>/dev/null
        vboxmanage unregistervm bootstrapper_$i --delete 2>/dev/null
    done
    for i in $(seq $WORKERS); do
        vboxmanage controlvm worker_$i poweroff 2>/dev/null
        vboxmanage unregistervm worker_$i --delete 2>/dev/null
    done
}

function startBootstrapperMachines {
    for i in $(seq $BOOTSTRAPPERS); do
        vboxmanage startvm bootstrapper_$i --type gui || true
    done
}

function initEtcd {
    docker rm -f blacksmith-dev-etcd 2>/dev/null || true
    docker run -d \
        -p 4001:4001 \
        -p 2380:2380 \
        -p 2379:2379 \
        --name blacksmith-dev-etcd quay.io/coreos/etcd:v2.2.3 \
        -name etcd0 \
        -advertise-client-urls http://${HostIP}:2379,http://${HostIP}:4001 \
        -listen-client-urls http://0.0.0.0:2379,http://0.0.0.0:4001 \
        -initial-advertise-peer-urls http://${HostIP}:2380 \
        -listen-peer-urls http://0.0.0.0:2380 \
        -initial-cluster-token etcd-cluster-1 \
        -initial-cluster etcd0=http://${HostIP}:2380 \
        -initial-cluster-state new
}

function runBlacksmith {
    docker rm -f blacksmith 2>/dev/null || true
    docker run -it --name blacksmith --restart=always --net=host \
      -v $current_workspace:/workspace \
      -v $(pwd)/certs/:/certs/ \
      -v /etc/ssl/certs/ca-certificates.crt:/etc/ssl/certs/ca-certificates.crt \
      quay.io/cafebazaar/blacksmith:v0.10 \
        -workspace /workspace \
        -etcd http://${HostIP}:2379 \
        -if $HOSTONLY \
        -cluster-name cafecluster \
        -lease-start 172.19.1.11 \
        -file-server http://${HostIP}/ \
        -lease-range 10 \
        -dns 8.8.8.8 \
        -debug \
        -http-listen ${HostIP}:8000 \
        -api-listen ${HostIP}:8001 \
        -tls-cert /certs/server.crt \
        -tls-key /certs/server.key \
        -tls-ca /certs/ca.crt \
        -workspace-repo https://github.com/cafebazaar/blacksmith-kubernetes
}


function runWithDelay {
    sleep $1
    "${@:2}"
}


#### clean
if [[ "${1:-}" == "clean" ]]; then

    removeMachines || true

    for i in 0 1 2 3 4 5 6 7 8 9; do
        if [[ ! -z $(VBoxManage list hostonlyifs | grep vboxnet$i) ]]; then
          vboxmanage hostonlyif remove vboxnet$i 2>/dev/null || true
        fi
    done

    rm .vbox_* 2>/dev/null
    rm -rf "$VMDIR"/worker_* 2>/dev/null
    rm -rf "$VMDIR"/bootstrapper_* 2>/dev/null

    docker rm -f blacksmith-dev-etcd || true
    docker rm -f blacksmith || true

    rm .vbox_cluster_inited 2>/dev/null || true

    echo "Cleaned."
    exit
fi


#### init bootstrappers etcd
if [[ "${1:-}" == "init-bootstrappers" ]]; then
    for i in $(seq $BOOTSTRAPPERS); do
        MAC=$(vboxmanage showvminfo bootstrapper_$i --machinereadable | grep macaddress1 | sed 's/macaddress1="\(.*\)"/\1/g')
        # FIXME: blacksmith-kubernetes specific thing
        # setDesiredState $MAC bootstrapper$i
        # vboxmanage controlvm bootstrapper_$i reset
    done
    exit
fi


#### init workers etcd
if [[ "${1:-}" == "init-workers" ]]; then
    for i in $(seq $WORKERS); do
        MAC=$(vboxmanage showvminfo worker_$i --machinereadable | grep macaddress2 | sed 's/macaddress2="\(.*\)"/\1/g')
        # FIXME: blacksmith-kubernetes specific thing
        # vboxmanage controlvm worker_$i reset
    done
    exit
fi


#### create workers
if [[ "${1:-}" == "worker" ]]; then
    if [[ "${2:-}" -eq "${2:-}" ]]; then
        WORKERS=$2
        echo $2 > .vbox_number_of_workers
    fi

    for i in $(seq $WORKERS); do
        MAC1="00025d13be8$((i*2))"
        MAC2="00025d17be8$((i*2))"
        create_machine worker_$i $MAC1 $MAC2

    done

    for i in $(seq $WORKERS); do
        vboxmanage startvm worker_$i --type gui &
    done

    runWithDelay 10 exec ./dev_run.sh init-workers &

    exit
fi


#### default run
make blacksmith 1>/dev/null

# check if blacksmith port is busy
if [[ ! -z $(netstat -lnt | awk '$6 == "LISTEN" && $4 ~ ".8000"') ]]; then
    echo "blacksmith port is already busy, perhaps another instance of it is open"
    exit 1
fi

if [[ ! -e ".vbox_cluster_inited" ]]
then
    createNetwork
    createBootstrappers
    initEtcd
    runWithDelay 10 exec ./dev_run.sh init-bootstrappers &
    touch .vbox_cluster_inited
fi
startBootstrapperMachines
# runWithDelay 5 xdg-open http://127.0.0.1:8000/ui &
runBlacksmith
