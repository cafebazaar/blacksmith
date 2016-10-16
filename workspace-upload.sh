#!/bin/bash
set -o errexit
set -o pipefail
set -o nounset
# set -o xtrace

IP=${1:-127.0.0.1}
MD5=($(md5sum ../blacksmith-kubernetes/workspace/files/workspace.tar))

curl -X POST -H "Content-Type: application/octet-stream" --data-binary '@../blacksmith-kubernetes/workspace/files/workspace.tar' http://127.0.0.1:8000/uploadworkspace/$MD5