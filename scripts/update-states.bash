#!/bin/bash
set -e
source ./scripts/common.bash

./blacksmithctl set node-key ${NODE1_MAC} state bootstrapper1
./blacksmithctl set node-key ${NODE2_MAC} state bootstrapper2
./blacksmithctl set node-key ${NODE3_MAC} state bootstrapper3

./blacksmithctl update workspaces
./blacksmithctl update node ${NODE1_MAC}
./blacksmithctl update node ${NODE2_MAC}
./blacksmithctl update node ${NODE3_MAC}

# curl -X POST http://${BobIP}:8000/api/update
# curl http://${BobIP}:8000/api/update
echo
