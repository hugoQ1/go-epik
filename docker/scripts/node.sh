#! /usr/bin/env bash

echo "env miner_init:${MINER_INIT}"

# cat ${HOME}/.epik/config.toml

make all
make install

command="epik daemon &"
# command="epik daemon > /var/log/epik-daemon.log 2>&1 &"
echo "Run $command"
eval $command


# command="epik-storage-miner run --nosync > /var/log/epik-miner.log 2>&1"
command="epik-storage-miner run --nosync"
if ${MINER_INIT}; then
    echo "execute miner init command:"
    command="epik-storage-miner init --nosync"
fi
echo "Run $command"
eval $command