#!/usr/bin/env bash

log() {
  echo -e "\e[33m$1\e[39m"
}

host=$1

log "> Deploying bootstrap node $host"
log "Stopping epik daemon"

log "Stopping epik daemon"

ssh "$host" 'systemctl stop epik-daemon' &
ssh "$host" 'systemctl stop epik-miner' &

wait

ssh "$host" 'rm -rf .epik' &
ssh "$host" 'rm -rf .epikminer' &

scp -C epik "${host}":/usr/local/bin/epik &
scp -C epik-miner "${host}":/usr/local/bin/epik-miner &

wait

log 'Initializing repo'

ssh "$host" 'systemctl start epik-daemon'
scp scripts/bootstrap.toml "${host}:.epik/config.toml"
ssh "$host" "echo -e '[Metrics]\nNickname=\"Boot-$host\"' >> .epik/config.toml"
ssh "$host" 'systemctl restart epik-daemon'

log 'Extracting addr info'

ssh "$host" 'epik net listen' | grep -v '/10' | grep -v '/127' >> build/bootstrap/bootstrappers.pi
