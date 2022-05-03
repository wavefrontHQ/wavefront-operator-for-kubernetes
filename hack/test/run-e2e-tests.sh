#!/bin/bash -e

REPO_ROOT=$(git rev-parse --show-toplevel)
source ${REPO_ROOT}/hack/test/k8s-utils.sh
source ${REPO_ROOT}/release/VERSION

function print_usage_and_exit() {
  echo "Failure: $1"
  echo "Usage: $0 [flags] [options]"
  echo -e "\t-c wavefront instance name (default: 'nimba')"
  echo -e "\t-t wavefront token (required)"
  echo -e "\t-n config cluster name for metric grouping (default: \$(whoami)-<default version from file>-release-test)"
  exit 1
}

function main() {

  # REQUIRED
  local WAVEFRONT_TOKEN=

  local WF_CLUSTER=nimba
  local VERSION=${COLLECTOR_VERSION}
  local K8S_ENV=$(cd ${REPO_ROOT}/hack/test && ./get-k8s-cluster-env.sh)
  local CONFIG_CLUSTER_NAME=$(whoami)-${K8S_ENV}-operator-$(date +"%y%m%d")

  while getopts ":c:t:v:n:p:" opt; do
    case $opt in
    c)
      WF_CLUSTER="$OPTARG"
      ;;
    t)
      WAVEFRONT_TOKEN="$OPTARG"
      ;;
    v)
      VERSION="$OPTARG"
      ;;
    n)
      CONFIG_CLUSTER_NAME="$OPTARG"
      ;;
    \?)
      print_usage_and_exit "Invalid option: -$OPTARG"
      ;;
    esac
  done

  if [[ -z ${WAVEFRONT_TOKEN} ]]; then
    print_usage_and_exit "wavefront token required"
  fi

  cd $REPO_ROOT
  sed "s/YOUR_CLUSTER_NAME/${CONFIG_CLUSTER_NAME}/g"  hack/test/_v1alpha1_wavefront_test.template.yaml  |
    sed "s/YOUR_WAVEFRONT_TOKEN/${WAVEFRONT_TOKEN}/g" > hack/test/_v1alpha1_wavefront_test.yaml

  kubectl apply -f hack/test/_v1alpha1_wavefront_test.yaml
  echo "Running test-wavefront-metrics"

  ${REPO_ROOT}/hack/test/test-wavefront-metrics.sh -t ${WAVEFRONT_TOKEN} -n ${CONFIG_CLUSTER_NAME}
  green "Success!"
}

main "$@"
