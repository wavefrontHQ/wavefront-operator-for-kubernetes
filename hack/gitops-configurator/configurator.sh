#!/usr/bin/env bash

REPO_DIR=./gitops-configurator
CONFIG_FILE=wavefront.yaml
POLL_INTERVAL=3

function init_repo_and_author() {
  git config --global user.name "Wavefront Operator Configurator"
  git config --global user.email "<>"
  if [ ! -d "$REPO_DIR" ]; then
    git clone "$CONFIG_REPO" "$REPO_DIR"
  fi
}

function push_cr() {
  kubectl get "$CR_SELECTOR" --namespace "$CR_NAMESPACE" -o yaml > "$CONFIG_FILE"

  git add "$CONFIG_FILE"
  git commit -m"Update '$CONFIG_FILE'" || return
  git push || return # should be a conflict if it was updated externally
}

function pull_cr() {
  git pull

  kubectl
}

function main() {
  init_repo_and_author

  cd "$REPO_DIR" || exit 1

  while [ true ]; do
    pull_cr
    push_cr
    sleep $POLL_INTERVAL
  done
}

main