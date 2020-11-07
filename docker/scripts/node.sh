#! /usr/bin/env bash

make all
sudo make install

command="epik daemon > /var/log/epik-daemon.log 2>&1 &"
echo "Run $command"
eval $command

command="epik-storage-miner run --nosync > /var/log/epik-miner.log 2>&1"
echo "Run $command"
eval $command