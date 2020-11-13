#!/usr/bin/env bash

set -euo pipefail
IFS=$'\n\t'


HOST=$1

# upload binaries
# TODO: destroy

FILES_TO_SEND=(
	./epik
	./epik-miner
	scripts/epik-daemon.service
	scripts/louts-miner.service
)

rsync -P "${FILES_TO_SEND[@]}" "$HOST:~/epik-stage/"

ssh "$HOST" 'bash -s' << 'EOF'
set -euo pipefail

systemctl stop epik-miner
systemctl stop epik-daemon
mkdir -p .epik .epikminer

cd "$HOME/epik-stage/"
cp -f epik epik-miner /usr/local/bin
cp -f epik-daemon.service /etc/systemd/system/epik-daemon.service
cp -f epik-miner.service /etc/systemd/system/epik-miner.service

systemctl daemon-reload
systemctl start epik-daemon
EOF


# setup miner actor
