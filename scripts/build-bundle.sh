#!/usr/bin/env bash
set -ex

ARCHS=(
    "darwin"
    "linux"
)

REQUIRED=(
    "ipfs"
    "sha512sum"
)
for REQUIRE in "${REQUIRED[@]}"
do
    command -v "${REQUIRE}" >/dev/null 2>&1 || echo >&2 "'${REQUIRE}' must be installed"
done

mkdir bundle
pushd bundle

BINARIES=(
    "epik"
    "epik-miner"
    "epik-worker"
)

export IPFS_PATH=`mktemp -d`
ipfs init
ipfs daemon &
PID="$!"
trap "kill -9 ${PID}" EXIT
sleep 30

for ARCH in "${ARCHS[@]}"
do
    mkdir -p "${ARCH}/epik"
    pushd "${ARCH}"
    for BINARY in "${BINARIES[@]}"
    do
        cp "../../${ARCH}/${BINARY}" "epik/"
        chmod +x "epik/${BINARY}"
    done

    tar -zcvf "../epik_${CIRCLE_TAG}_${ARCH}-amd64.tar.gz" epik
    popd
    rm -rf "${ARCH}"

    sha512sum "epik_${CIRCLE_TAG}_${ARCH}-amd64.tar.gz" > "epik_${CIRCLE_TAG}_${ARCH}-amd64.tar.gz.sha512"

    ipfs add -q "epik_${CIRCLE_TAG}_${ARCH}-amd64.tar.gz" > "epik_${CIRCLE_TAG}_${ARCH}-amd64.tar.gz.cid"
done
popd
