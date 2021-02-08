#!/usr/bin/env bash

session="epik-interop"
wdaemon="daemon"
wminer="miner"
wsetup="setup"
wpledging="pledging"
wcli="cli"
faucet="https://t01000.miner.interopnet.kittyhawk.wtf"

PLEDGE_COUNT="${1:-20}"
BRANCH="interopnet"
BASEDIR=$(mktemp -d -t "epik-interopnet.XXXX")

git clone --branch "$BRANCH" https://github.com/EpiK-Protocol/go-epik.git "${BASEDIR}/build"

mkdir -p "${BASEDIR}/scripts"
mkdir -p "${BASEDIR}/bin"

cat > "${BASEDIR}/scripts/build.bash" <<EOF
#!/usr/bin/env bash
set -x

SCRIPTDIR="\$( cd "\$( dirname "\${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
pushd \$SCRIPTDIR/../build

pwd
env RUSTFLAGS="-C target-cpu=native -g" FFI_BUILD_FROM_SOURCE=1 make clean deps epik epik-miner epik-shed
cp epik epik-miner epik-shed ../bin/

popd
EOF

cat > "${BASEDIR}/scripts/env.fish" <<EOF
set -x PATH ${BASEDIR}/bin \$PATH
set -x EPIK_PATH ${BASEDIR}/.epik
set -x EPIK_MINER_PATH ${BASEDIR}/.epikminer
EOF

cat > "${BASEDIR}/scripts/env.bash" <<EOF
export PATH=${BASEDIR}/bin:\$PATH
export EPIK_PATH=${BASEDIR}/.epik
export EPIK_MINER_PATH=${BASEDIR}/.epikminer
EOF

cat > "${BASEDIR}/scripts/create_miner.bash" <<EOF
#!/usr/bin/env bash
set -x

owner=\$(epik wallet new bls)
result=\$(curl -D - -XPOST -F "sectorSize=536870912" -F "address=\$owner" $faucet/mkminer | grep Location)
query_string=\$(grep -o "\bf=.*\b" <<<\$(echo \$result))

declare -A param
while IFS='=' read -r -d '&' key value && [[ -n "\$key" ]]; do
    param["\$key"]=\$value
done <<<"\${query_string}&"

epik state wait-msg "\${param[f]}"

maddr=\$(curl "$faucet/msgwaitaddr?cid=\${param[f]}" | jq -r '.addr')

epik-miner init --actor=\$maddr --owner=\$owner
EOF

cat > "${BASEDIR}/scripts/pledge_sectors.bash" <<EOF
#!/usr/bin/env bash

set -x

while [ ! -d ${BASEDIR}/.epikminer ]; do
  sleep 5
done

while [ ! -f ${BASEDIR}/.epikminer/api ]; do
  sleep 5
done

sleep 30

sector=\$(epik-miner sectors list | tail -n1 | awk '{print \$1}' | tr -d ':')
current="\$sector"

while true; do
  if (( \$(epik-miner sectors list | wc -l) > ${PLEDGE_COUNT} )); then
    break
  fi

  while true; do
    state=\$(epik-miner sectors list | tail -n1 | awk '{print \$2}')

    if [ -z "\$state" ]; then
      break
    fi

    case \$state in
      PreCommit1 | PreCommit2 | Packing | Unsealed | PreCommitting | Committing | CommitWait | FinalizeSector ) sleep 30 ;;
      WaitSeed | Proving ) break ;;
      * ) echo "Unknown Sector State: \$state"
          epik-miner sectors status --log \$current
          break ;;
    esac
  done

  epik-miner sectors pledge

  while [ "\$current" == "\$sector" ]; do
    sector=\$(epik-miner sectors list | tail -n1 | awk '{print \$1}' | tr -d ':')
    sleep 5
  done

  current="\$sector"
done
EOF

cat > "${BASEDIR}/scripts/monitor.bash" <<EOF
#!/usr/bin/env bash

while true; do
  clear
  epik sync status

  echo
  echo
  echo Miner Info
  epik-miner info

  echo
  echo
  echo Sector List
  epik-miner sectors list | tail -n4

  sleep 25

  epik-shed noncefix --addr \$(epik wallet list) --auto

done
EOF

chmod +x "${BASEDIR}/scripts/build.bash"
chmod +x "${BASEDIR}/scripts/create_miner.bash"
chmod +x "${BASEDIR}/scripts/pledge_sectors.bash"
chmod +x "${BASEDIR}/scripts/monitor.bash"

bash "${BASEDIR}/scripts/build.bash"

tmux new-session -d -s $session -n $wsetup

tmux set-environment -t $session BASEDIR "$BASEDIR"

tmux new-window -t $session -n $wcli
tmux new-window -t $session -n $wdaemon
tmux new-window -t $session -n $wminer
tmux new-window -t $session -n $wpledging

tmux kill-window -t $session:$wsetup

case $(basename $SHELL) in
  fish ) shell=fish ;;
  *    ) shell=bash ;;
esac

tmux send-keys -t $session:$wdaemon   "source ${BASEDIR}/scripts/env.$shell" C-m
tmux send-keys -t $session:$wminer    "source ${BASEDIR}/scripts/env.$shell" C-m
tmux send-keys -t $session:$wcli      "source ${BASEDIR}/scripts/env.$shell" C-m
tmux send-keys -t $session:$wpledging "source ${BASEDIR}/scripts/env.$shell" C-m

tmux send-keys -t $session:$wdaemon "epik daemon --api 48010 daemon 2>&1 | tee -a ${BASEDIR}/daemon.log" C-m

sleep 30

tmux send-keys -t $session:$wminer   "${BASEDIR}/scripts/create_miner.bash" C-m
tmux send-keys -t $session:$wminer   "epik-miner run --api 48020 2>&1 | tee -a ${BASEDIR}/miner.log" C-m
tmux send-keys -t $session:$wcli     "${BASEDIR}/scripts/monitor.bash" C-m
tmux send-keys -t $session:$wpleding "${BASEDIR}/scripts/pledge_sectors.bash" C-m

tmux select-window -t $session:$wcli

tmux attach-session -t $session

