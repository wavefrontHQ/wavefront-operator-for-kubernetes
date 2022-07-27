#!/bin/bash -e

REPO_ROOT=$(git rev-parse --show-toplevel)
source ${REPO_ROOT}/hack/test/k8s-utils.sh

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
  local WAVEFRONT_URL="https://nimba.wavefront.com"
  local WF_CLUSTER=nimba
  local CONFIG_CLUSTER_NAME=$(create_cluster_name)

  while getopts ":c:t:n:p:" opt; do
    case $opt in
    c)
      WF_CLUSTER="$OPTARG"
      ;;
    t)
      WAVEFRONT_TOKEN="$OPTARG"
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


  helm repo add wavefront-v2beta https://projects.registry.vmware.com/chartrepo/tanzu_observability
  helm repo update
  kubectl create namespace wavefront || true
  helm uninstall wavefront-v2beta -n wavefront || true
  helm install wavefront-v2beta wavefront-v2beta/wavefront-v2beta --namespace wavefront || true
  kubectl create -n wavefront secret generic wavefront-secret --from-literal token=${WAVEFRONT_TOKEN} || true

  cat <<EOF | kubectl apply -f -
  apiVersion: wavefront.com/v1alpha1
  kind: Wavefront
  metadata:
    name: wavefront
    namespace: wavefront
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
  kubectl get wavefront -n wavefront
}

main "$@"
