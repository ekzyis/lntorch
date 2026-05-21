#!/usr/bin/env bash
#
# This script is called by the GitHub actions runner,
# see .github/workflows/deploy.yml.

set -e

PNAME=lntorch
PORT=4444

COMMIT="${SSH_ORIGINAL_COMMAND:-HEAD}"
if [[ ! "$COMMIT" =~ ^[a-f0-9]{40}$ ]]; then
    echo "Invalid commit SHA"
    exit 1
fi

nix-shell -p figlet --run "figlet -f smslant $PNAME"
echo "deploying commit $COMMIT"

set -x

cd $PNAME
git fetch
git switch --detach "$COMMIT"
go build -o $PNAME
tmux kill-session -t $PNAME || true
tmux new-session -d -s $PNAME "./$PNAME $PORT"
