#!/usr/bin/env bash

HOST=$1

scp scripts/epik-daemon.service "${HOST}:/etc/systemd/system/epik-daemon.service"
scp scripts/epik-miner.service "${HOST}:/etc/systemd/system/epik-miner.service"
