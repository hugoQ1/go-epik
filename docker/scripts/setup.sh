#!/bin/bash

process=$PROCESS
coinbase=$COINBASE

if [ $# -gt 0 ]; then
    process=$1
fi

if [ $# -gt 1 ]; then
    coinbase=$2
fi

echo $process

# git config --global user.name "epik"
# git config --global user.email "you@example.com"
# git pull --rebase
make all
make install
echo "Update to latest Node!!!"
echo "FULLNODE_API_INFO:$FULLNODE_API_INFO"

if [ "$process" = "daemon" ]; then
    unset FULLNODE_API_INFO
    if [ -d /root/.epik/datastore ]; then
        epik daemon
    else
        epik daemon --import-snapshot https://epik.obs.cn-southwest-2.myhuaweicloud.com/snapshots/latest.car
    fi
elif [ "$process" = "miner" ]; then
    if [ -f /root/.epikminer/minerid ]; then
        epik-miner run
    else
        if [ "x$coinbase" = "x" ]; then
            epik-miner init
        else
            epik-miner init --coinbase $coinbase
        fi
    fi
else
    echo "ERROR: unsupport process!"
    exit 1
fi