#!/bin/bash

if ! which vboxmanage > /dev/null; then
    echo 'You should install VirtualBox or make sure "vboxmanage" is in path'
    exit 1
fi

if [[ ! $(vboxmanage list extpacks | grep "Oracle VM VirtualBox Extension Pack") ]]; then
    echo 'You should make sure if "virtualbox-ext-pack" is installed which is needed for PXE boot thing, its installation however could be tricky (needs proxy)'
    exit 1
fi


####
# BoB IP
HostIP="192.168.56.1"
# hostonly network name, it is "vboxnet0" by default and we have less control for what it should be it seems
HOSTONLY=$(cat .vbox_network_hostonly_if) || "vboxnet0"
# NAT network interface name
NATNAME="NatNetwork"
# detects which interface connects us to the Internet, needed for bridge
INTERTNETIF=$(route | grep '^default' | grep -o '[^ ]*$')
# number of bootstrapper, 3 is good enough usually
BOOTSTAPPERS=3
# number of workers
WORKERS=$(cat .vbox_number_of_workers) || "3"


####
function create_network {
    # usually it installs a hostonly network named "vboxnet0", it would be nice if we could control its name, it seems we can't, however
    HOSTONLY=$(vboxmanage hostonlyif create 2>/dev/null | sed "s/.*'\(.*\)'.*/\1/g")
    echo $HOSTONLY > .vbox_network_hostonly_if
    # vboxmanage hostonlyif ipconfig $HOSTONLY --ip192.168.56.1

    vboxmanage natnetwork add --netname $NATNAME --dhcp off --network "172.19.1.0/24" --enable
}

function create_machine {
    vboxmanage createvm --name $1 --register

    vboxmanage modifyvm $1 \
        --ostype "Linux_64" \
        --memory 2048 \
        --nic1 hostonly \
        --nictype1 82540EM \
        --hostonlyadapter1 $HOSTONLY \
        --nicpromisc1 allow-all \
        --nic2 natnetwork \
        --natnet2 $NATNAME \
        --nicpromisc2 allow-all \
        --nic3 bridged \
        --bridgeadapter3 $INTERTNETIF \
        --nicpromisc3 allow-all \
        --boot1 disk \
        --boot2 net \
        --boot3 none \
        --boot4 none

    vboxmanage storagectl $1 \
        --name IDE0 \
        --add ide

    vboxmanage createhd \
        --filename $1 \
        --size 8000 \
        --variant Standard

    vboxmanage storageattach $1 \
        --storagectl IDE0 \
        --port 0 \
        --device 0 \
        --type hdd \
        --medium $1.vdi
}

function create_bootstrappers {
    for i in $(seq $BOOTSTAPPERS); do
        create_machine bootstrapper_$i
    done
}

function remove_machines {
    for i in $(seq $BOOTSTAPPERS); do
        vboxmanage controlvm bootstrapper_$i poweroff
        vboxmanage unregistervm bootstrapper_$i --delete
    done
    for i in $(seq $WORKERS); do
        vboxmanage controlvm worker_$i poweroff
        vboxmanage unregistervm worker_$i --delete
    done
}

function start_bootstrapper_machines {
    for i in $(seq $BOOTSTAPPERS); do
        vboxmanage startvm bootstrapper_$i --type gui
    done
}

function init_etcd {
    docker rm -f blacksmith-dev-etcd
    docker run -d \
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

function run_blacksmith {
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


#### clean
if [ "$1" == "clean" ]; then
    remove_machines
    vboxmanage hostonlyif remove $HOSTONLY
    vboxmanage natnetwork remove --netname $NATNAME
    rm .vbox_*
    rm *.vdi
    rm -rf ~/VirtualBox\ VMs/worker_*
    rm -rf ~/VirtualBox\ VMs/bootstrapper_*
    echo "Cleaned."
    exit
fi


#### create workers
if [ "$1" == "worker" ]; then
    if [ "$2" -eq "$2" ]; then
        WORKERS=$2
        echo $2 > .vbox_number_of_workers
    fi

    for i in $(seq $WORKERS); do
        create_machine worker_$i
    done

    for i in $(seq $WORKERS); do
        vboxmanage startvm worker_$i --type gui &
    done

    exit
fi


#### initialize state of machines
if [ "$1" == "init" ]; then
    #FIXME: These are blacksmith-kubernetes specific thigs indeed and should be moved there 
    for i in $(seq $BOOTSTAPPERS); do
        MAC=$(vboxmanage showvminfo bootstrapper_$i --machinereadable | grep macaddress1 | sed 's/macaddress1="\(.*\)"/\1/g')
        curl -X PUT "http://localhost:2379/v2/keys/cafecluster/machines/$MAC/desired-state?value=bootstrapper$i"
        curl -X PUT "http://localhost:2379/v2/keys/cafecluster/machines/$MAC/state?value=init-install-coreos"
        vboxmanage controlvm bootstrapper_$i reset
    done
    for i in $(seq $WORKERS); do
        MAC=$(vboxmanage showvminfo worker_$i --machinereadable | grep macaddress1 | sed 's/macaddress1="\(.*\)"/\1/g')
        curl -X PUT "http://localhost:2379/v2/keys/cafecluster/machines/$MAC/state?value=init-worker"
        vboxmanage controlvm worker_$i reset
    done

    exit
fi

#### default run
make blacksmith 1>/dev/null 2>&1
if [ ! -e ".vbox_network_inited" ]; then create_network; touch .vbox_network_inited; fi
if [ ! -e ".vbox_bootstrappers_inited" ]; then create_bootstrappers; touch .vbox_bootstrappers_inited; fi
init_etcd
start_bootstrapper_machines
xdg-open http://127.0.0.1:8000/ui
run_blacksmith
