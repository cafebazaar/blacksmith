# Network
Bridge1=blacksmith0
Bridge2=blacksmith1
BobIP=172.20.0.1
Subnet1=172.20.0.1/24
Subnet2=172.21.0.1/24
LeaseStart=172.20.0.11
InternetInterface=$(ip route get 8.8.8.8 | head -1 | cut -d " " -f 5)

NODE1_NAME=node1
NODE1_MAC=00:02:7d:15:be:82
NODE2_NAME=node2
NODE2_MAC=00:02:7d:15:be:84
NODE3_NAME=node3
NODE3_MAC=00:02:7d:15:be:86
NODES=(node1 node2 node3)

COREOS_VERSION=1248.4.0
COREOS_CHANNEL=beta

# Common config

# BlacksmithImageForBob=localhost:5000/blacksmith:dev
# BlacksmithImage=${BobIP}:5000/blacksmith:dev
BlacksmithImageForBob=quay.io/cafebazaar/blacksmith:dev
BlacksmithImage=quay.io/cafebazaar/blacksmith:dev

# WorkspaceGitURL=git://${BobIP}/for-refactoring/.git
WorkspaceGitURL=git@git.cafebazaar.ir:ali.javadi/blacksmith-kubernetes.git
WorkspaceGitBranch=dev

# BlacksmithImageForBob=quay.io/cafebazaar/blacksmith:dev
# BlacksmithImage=quay.io/cafebazaar/blacksmith:dev
# WorkspaceGitURL=git@git.cafebazaar.ir:ali.javadi/blacksmith-kubernetes.git
