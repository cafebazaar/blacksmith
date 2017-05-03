set -x
./blacksmithctl update workspaces

./blacksmithctl set node-key 00:02:7d:15:be:82 state bootstrapper1
./blacksmithctl set node-key 00:02:7d:15:be:84 state bootstrapper2
./blacksmithctl set node-key 00:02:7d:15:be:86 state bootstrapper3

./blacksmithctl install node 00:02:7d:15:be:82
./blacksmithctl install node 00:02:7d:15:be:84
./blacksmithctl install node 00:02:7d:15:be:86
