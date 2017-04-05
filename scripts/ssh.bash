#!/bin/bash
# Manage VM nodes which have a specific set of hardware attributes.

# Example:
#   $ bash scripts/ssh.bash 172.20.0.11

function main {
  local SSH_OPTS="StrictHostKeyChecking no"
  IP=$1
  shift
  if [ -z "$IP" ]; then
    usage
    exit 1
  fi

  # TODO: enable this by a flag
  ssh-keygen -f ~/.ssh/known_hosts -R $IP > /dev/null

  if [ -z "$1" ]; then
    ssh -o "$SSH_OPTS" core@$IP
  else
    ssh -o "$SSH_OPTS" core@$IP -t "$@"
  fi
}

function usage {
  echo "USAGE: $0 <ip>"
}

main "$@"
