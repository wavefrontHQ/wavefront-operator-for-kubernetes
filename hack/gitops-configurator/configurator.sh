#!/usr/bin/env bash

REPO_DIR=./gitops-configurator
CONFIG_FILE=wavefront.yaml
SLEEP_INTERVAL=60

function init_repo() {
    if [ ! -d "$REPO_DIR" ]; then
        git clone "$CONFIG_REPO" "$REPO_DIR"
    fi
}

function main() {
  init_repo



  while [ true ]; do
    kubectl get "$CR_SELECTOR" --namespace "$CR_NAMESPACE" -o yaml > "$REPO_DIR/$CONFIG_FILE"

    pushd "$REPO_DIR" || exit 1
      git add "$CONFIG_FILE"
      git commit -m"Update '$CONFIG_FILE'" || continue
      git push
    popd  || exit 1

    sleep $SLEEP_INTERVAL
  done
}
