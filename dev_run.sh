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
HostIP="192.168.56.1"
# hostonly network name, it is "vboxnet0" by default and we have less control for what it should be it seems
HOSTONLY="vboxnet0"
if [[ -f ".vbox_network_hostonly_if" ]]; then HOSTONLY=$(cat .vbox_network_hostonly_if); fi
# NAT network interface name
NATNAME="NatNetwork"
# detects which interface connects us to the Internet, needed for bridge
INTERTNETIF=$(route | grep '^default' | grep -o '[^ ]*$')
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
    # vboxmanage hostonlyif ipconfig $HOSTONLY --ip192.168.56.1

    vboxmanage natnetwork add --netname $NATNAME --dhcp off --network "172.19.1.0/24" --enable
}

function create_machine {
    vboxmanage createvm --name $1 --ostype "Linux_64"

    vboxmanage registervm "$VMDIR/$1/$1.vbox"

    vboxmanage modifyvm $1 \
        --memory 2048 \
        --nic1 hostonly \
        --nictype1 82540EM \
        --hostonlyadapter1 $HOSTONLY \
        --nicpromisc1 allow-all \
        --nic2 natnetwork \
        --nictype2 82540EM \
        --natnet2 $NATNAME \
        --nicpromisc2 allow-all \
        --nic3 bridged \
        --nictype3 82540EM \
        --bridgeadapter3 $INTERTNETIF \
        --nicpromisc3 allow-all \
        --boot1 disk \
        --boot2 net \
        --boot3 none \
        --boot4 none

    vboxmanage storagectl $1 \
        --name IDE0 \
        --add sata \
        --bootable on \
        2>/dev/null

    vboxmanage createhd \
        --filename "$VMDIR/$1/$1" \
        --size 8000

    vboxmanage storageattach $1 \
        --storagectl IDE0 \
        --port 0 \
        --device 0 \
        --type hdd \
        --medium "$VMDIR/$1/$1.vdi"
}

function createBootstrappers {
    for i in $(seq $BOOTSTRAPPERS); do
        create_machine bootstrapper_$i
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
    sudo docker rm -f blacksmith-dev-etcd 2>/dev/null || true
    sudo docker run -d \
        -p 4001:4001 \
        -p 2380:2380 \
        -p 2379:2379 \
        --name blacksmith-dev-etcd quay.io/coreos/etcd:v2.2.4 \
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
    sudo ./blacksmith \
        -workspace $(pwd)/workspaces/current \
        -etcd http://${HostIP}:2379 \
        -if $HOSTONLY \
        -cluster-name cafecluster \
        -lease-start 192.168.56.20 \
        -lease-range 10 \
        -dns 8.8.8.8 \
        -debug \
        -http-listen :8000
}

function setState {
    # ",," means converting a string to lowercase
    local MAC=${1,,}
    local VALUE=$2
    curl -X PUT "http://localhost:2379/v2/keys/cafecluster/machines/$MAC/state?value=$VALUE"
}

function setInternalState {
    # ",," means converting a string to lowercase
    local MAC=${1,,}
    local VALUE=$2
    for i in $(seq $BOOTSTRAPPERS); do
        local MASTERMAC=$(vboxmanage showvminfo bootstrapper_$i --machinereadable | grep macaddress1 | sed 's/macaddress1="\(.*\)"/\1/g')
        local MASTERMAC=$(echo ${MASTERMAC} | sed -e 's/[0-9A-F]\{2\}/&:/g' -e 's/:$//')
        local IPS=$(ip neighbor | grep -i "${MASTERMAC}" | cut -d" " -f1)
    
        for ip in $(echo $IPS); do
            ssh-keygen -R $ip &> /dev/null
            ssh -o StrictHostKeyChecking=no core@$ip "curl -X PUT \"http://localhost:2379/v2/keys/cafecluster/machines/$MAC/state?value=$VALUE\" &>/dev/null" &>/dev/null || true
        done
    done
}

function setDesiredState {
    local MAC=${1,,}
    local VALUE=$2
    curl -X PUT "http://localhost:2379/v2/keys/cafecluster/machines/$MAC/desired-state?value=$VALUE"
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

    vboxmanage natnetwork remove --netname $NATNAME 2>/dev/null
    rm .vbox_* 2>/dev/null
    rm -rf "$VMDIR"/worker_* 2>/dev/null
    rm -rf "$VMDIR"/bootstrapper_* 2>/dev/null

    sudo docker rm -f blacksmith-dev-etcd || true

    rm .vbox_cluster_inited 2>/dev/null || true

    echo "Cleaned."
    exit
fi


#### upload workspace
if [[ "${1:-}" == "upload-workspace" ]]; then
    for i in $(seq $BOOTSTRAPPERS); do
        MAC=$(vboxmanage showvminfo bootstrapper_$i --machinereadable | grep macaddress3 | sed 's/macaddress3="\(.*\)"/\1/g')
        MAC=$(echo $MAC | sed -e 's/[0-9A-F]\{2\}/&:/g' -e 's/:$//')
        IP=$(ip neighbor | grep -i "${MAC}" | cut -d" " -f1)
        echo "Starting $IP"
        ./workspace-upload.sh $IP
    done
    exit
fi


#### init bootstrappers etcd
if [[ "${1:-}" == "init-bootstrappers" ]]; then
    for i in $(seq $BOOTSTRAPPERS); do
        MAC=$(vboxmanage showvminfo bootstrapper_$i --machinereadable | grep macaddress1 | sed 's/macaddress1="\(.*\)"/\1/g')
        # FIXME: blacksmith-kubernetes specific thing
        setDesiredState $MAC bootstrapper$i
        setState $MAC init-install-coreos
        vboxmanage controlvm bootstrapper_$i reset
    done
    exit
fi


#### init workers etcd
if [[ "${1:-}" == "init-workers" ]]; then
    for i in $(seq $WORKERS); do
        MAC=$(vboxmanage showvminfo worker_$i --machinereadable | grep macaddress2 | sed 's/macaddress2="\(.*\)"/\1/g')
        # FIXME: blacksmith-kubernetes specific thing
        setInternalState $MAC init-worker
        vboxmanage controlvm worker_$i reset
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
        create_machine worker_$i
        vboxmanage modifyvm worker_$i --nic1 none
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

# dummy command to make sure next sudo will be ran without delay
sudo echo

if [[ ! -e ".vbox_cluster_inited" ]]
then
    createNetwork
    createBootstrappers
    initEtcd
    sudo rm -rf workspaces/*
    runWithDelay 10 exec ./workspace-upload.sh &
    runWithDelay 15 exec ./dev_run.sh init-bootstrappers &
    touch .vbox_cluster_inited
fi
startBootstrapperMachines
runWithDelay 5 xdg-open http://127.0.0.1:8000/ui &
runBlacksmith
