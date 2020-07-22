#!/usr/bin/env bash

HOST=$1

ssh "$HOST" '[ -e ~/.epikstorage/token ]' && exit 0

ssh "$HOST" 'epik wallet new bls > addr'
ssh "$HOST" 'curl http://147.75.80.29:777/sendcoll?address=$(cat addr)' &
ssh "$HOST" 'curl http://147.75.80.29:777/sendcoll?address=$(cat addr)' &
ssh "$HOST" 'curl http://147.75.80.29:777/send?address=$(cat addr)' &
wait

echo "SYNC WAIT"
sleep 30

ssh "$HOST" 'epik sync wait'
ssh "$HOST" 'epik-storage-miner init --owner=$(cat addr)'
ssh "$HOST" 'systemctl start epik-storage-miner' &
