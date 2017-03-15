#### #!/bin/bash
#### set -e
#### source ./scripts/common.bash
#### 
#### function main {
####   case "$1" in
####     "create-docker") create-docker;;
####     "destroy") destroy;;
####     *)
####       echo "unknown command"
####       exit 2
####       ;;
####   esac
#### }
#### 
#### function create-docker {
####   local OPTS="--memory=1024 --vcpus=1 --pxe --disk pool=default,size=6 --os-type=linux --os-variant=generic --noautoconsole --events on_poweroff=preserve"
####   virt-install --name $NODE1_NAME --network=bridge:$Bridge,mac=$NODE1_MAC $OPTS --boot=hd,network
####   virt-install --name $NODE2_NAME --network=bridge:$Bridge,mac=$NODE2_MAC $OPTS --boot=hd,network
####   virt-install --name $NODE3_NAME --network=bridge:$Bridge,mac=$NODE3_MAC $OPTS --boot=hd,network
#### }
#### 
#### function destroy {
####   for node in ${NODES[@]}; do
####     virsh destroy $node
####   done
####   for node in ${NODES[@]}; do
####     virsh undefine $node
####   done
####   virsh pool-refresh default
####   for node in ${NODES[@]}; do
####     virsh vol-delete --pool default $node.qcow2
####   done
#### }
#### 
#### main $@
