# Network
Bridge=blacksmith0
BobIP=172.20.0.1
Subnet=172.20.0.1/24
LeaseStart=172.20.0.11

NODE1_NAME=node1
NODE1_MAC=00:02:7d:15:be:82
NODE2_NAME=node2
NODE2_MAC=00:02:7d:15:be:84
NODE3_NAME=node3
NODE3_MAC=00:02:7d:15:be:86
NODES=(node1 node2 node3)

## NODE1_NAME=node1
## NODE1_MAC=52:54:00:a1:9c:ae
## NODE2_NAME=node2
## NODE2_MAC=52:54:00:b2:2f:86
## NODE3_NAME=node3
## NODE3_MAC=52:54:00:c3:61:77

# Common config
BlacksmithImageForBob=localhost:5000/blacksmith:refactoring
BlacksmithImage=${BobIP}:5000/blacksmith:refactoring
WorkspaceGitURL=git://${BobIP}/for-refactoring/.git

# BlacksmithImageForBob=quay.io/cafebazaar/blacksmith:dev
# BlacksmithImage=quay.io/cafebazaar/blacksmith:dev
# WorkspaceGitURL=git@git.cafebazaar.ir:ali.javadi/blacksmith-kubernetes.git
