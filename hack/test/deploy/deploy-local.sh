#!/bin/bash -e

REPO_ROOT=$(git rev-parse --show-toplevel)
source ${REPO_ROOT}/hack/test/k8s-utils.sh

function print_usage_and_exit() {
  echo "Failure: $1"
  echo "Usage: $0 [flags] [options]"
  echo -e "\t-c wavefront instance name (default: 'nimba')"
  echo -e "\t-t wavefront token (required)"
  echo -e "\t-l wavefront logging token (required)"
  echo -e "\t-n config cluster name for metric grouping (default: \$(whoami)-<default version from file>-release-test)"
  exit 1
}

function main() {

  # REQUIRED
  local WAVEFRONT_TOKEN=
  local WAVEFRONT_LOGGING_TOKEN=
  local WAVEFRONT_URL="https://nimba.wavefront.com"
  local WF_CLUSTER=nimba
  local CONFIG_CLUSTER_NAME=$(create_cluster_name)

  while getopts ":c:t:l:n:p:" opt; do
    case $opt in
    c)
      WF_CLUSTER="$OPTARG"
      ;;
    t)
      WAVEFRONT_TOKEN="$OPTARG"
      ;;
    l)
      WAVEFRONT_LOGGING_TOKEN="$OPTARG"
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


  kubectl delete -f ./deploy/kubernetes/wavefront-operator.yaml || true
  kubectl apply -f ./deploy/kubernetes/wavefront-operator.yaml
  kubectl create -n observability-system secret generic wavefront-secret --from-literal token=${WAVEFRONT_TOKEN} || true
  kubectl create -n observability-system secret generic wavefront-secret-logging --from-literal token=${WAVEFRONT_LOGGING_TOKEN} || true

  cat <<EOF | kubectl apply -f -
  apiVersion: wavefront.com/v1alpha1
  kind: Wavefront
  metadata:
    name: wavefront
    namespace: observability-system
  spec:
    clusterName: $CONFIG_CLUSTER_NAME
    wavefrontUrl: $WAVEFRONT_URL
    dataCollection:
      metrics:
        enable: true
    dataExport:
      wavefrontProxy:
        enable: true
EOF

  wait_for_cluster_ready
  kubectl get wavefront -n observability-system
}

main "$@"
