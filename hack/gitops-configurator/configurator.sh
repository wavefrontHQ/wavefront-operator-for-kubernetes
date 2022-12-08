#!/usr/bin/env bash

REPO_DIR=./gitops-configurator
CONFIG_FILE=wavefront.yaml
SLEEP_INTERVAL=60

function init_repo_and_author() {
  git config --global user.name "Wavefront Operator Configurator"
  git config --global user.email "<>"
  if [ ! -d "$REPO_DIR" ]; then
    git clone "$CONFIG_REPO" "$REPO_DIR"
  fi
}

function push_cr() {
  while [ true ]; do
    kubectl get pods --namespace "$CR_NAMESPACE"
    kubectl get "$CR_SELECTOR" --namespace "$CR_NAMESPACE" -o yaml > "$REPO_DIR/$CONFIG_FILE"

    pushd "$REPO_DIR" || exit 1
      git add "$CONFIG_FILE"
      git commit -m"Update '$CONFIG_FILE'" || continue
      git push
    popd  || exit 1

    sleep $SLEEP_INTERVAL
  done
}

function main() {
  init_repo_and_author

  push_cr
}

main