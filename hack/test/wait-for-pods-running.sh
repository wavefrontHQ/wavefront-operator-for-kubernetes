#!/bin/bash -e

REPO_ROOT=$(git rev-parse --show-toplevel)
source ${REPO_ROOT}/hack/test/k8s-utils.sh

function main() {
  wait_for_cluster_ready
}

#TODO: Figure out the difference between READY vs RUNNING state. Below is WIP
#function wait_for_cluster_running() {
#  echo "Waiting for all Pods to be 'Running'"
#  while ! kubectl wait --for=jsonpath='{.status.phase}'=Running pod --all -l exclude-me!=true --all-namespaces &> /dev/null; do
#    echo "Waiting for all Pods to be 'Running'"
#    sleep 5
#  done
#  echo "All Pods are Running"
#}

main "$@"
