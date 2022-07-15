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

function run_test() {
  local type=$1
  local need_to_run_static_analysis="${2:-false}"
  local cluster_name=${CONFIG_CLUSTER_NAME}-$type

  echo "Running $type CR"

  wait_for_cluster_ready

  sed "s/YOUR_CLUSTER_NAME/$cluster_name/g"  ${REPO_ROOT}/hack/test/deploy/scenarios/wavefront-$type.yaml  |
   sed "s/YOUR_WAVEFRONT_URL/${WAVEFRONT_URL}/g" > hack/test/_v1alpha1_wavefront_test.yaml

  kubectl apply -f hack/test/_v1alpha1_wavefront_test.yaml

  wait_for_cluster_ready

  echo "Running test-wavefront-metrics"
  ${REPO_ROOT}/hack/test/test-wavefront-metrics.sh -t ${WAVEFRONT_TOKEN} -n $cluster_name -v ${COLLECTOR_VERSION} -e "$type-test.sh"

  if "$need_to_run_static_analysis"; then
    run_static_analysis
  fi;

  green "Success!"
  kubectl delete -f hack/test/_v1alpha1_wavefront_test.yaml
}

function run_static_analysis() {
  echo "Running static analysis"
  rm -rf kube-lint-results.txt
  kubectl api-resources --verbs=list --namespaced -o name \
  | xargs -n1 -I{} bash -c "kubectl get {} -n wavefront -oyaml && echo ---" \
  | ${REPO_ROOT}/bin/kube-linter lint - 1>kube-lint-results.txt 2>&1 || true

  # pods and replica sets are just a duplicate of deployment and daemon sets. So removing them before calculating error count
  local current_lint_errors=$(grep '<standard input>' kube-lint-results.txt | grep -v 'Kind=Pod' | grep -v 'Kind=ReplicaSet' | wc -l)
  yellow "Kube linter error count: ${current_lint_errors}"
  local known_lint_errors=2
  if [ $current_lint_errors -gt $known_lint_errors ]; then
    red "Failure: Found $(($current_lint_errors-$known_lint_errors)) newer error(s) more than the previously known ${known_lint_errors} error(s)"
    grep '<standard input>' kube-lint-results.txt | grep -v 'Kind=Pod' | grep -v 'Kind=ReplicaSet'
    exit_status=1
  fi

  rm -rf kube-score-results.txt
  kubectl api-resources --verbs=list --namespaced -o name \
  | xargs -n1 -I{} bash -c "kubectl get {} -n wavefront -oyaml && echo ---" \
  | ${REPO_ROOT}/bin/kube-score score - --output-format ci> kube-score-results.txt || true

  # pods are just a duplicate of deployment and daemon sets. So removing them before calculating error count
  local current_score_errors=$(grep '\[CRITICAL\]' kube-score-results.txt | grep -v 'v1/Pod' | wc -l)
  yellow "Kube score error count: ${current_score_errors}"
  local known_score_errors=6
  if [ $current_score_errors -gt $known_score_errors ]; then
    red "Failure: Found $(($current_score_errors-$known_score_errors)) newer error(s) more than the previously known ${known_score_errors} error(s)"
    grep '\[CRITICAL\]' kube-score-results.txt | grep -v 'v1/Pod'
    exit_status=1
  fi
  exit "${exit_status:-0}"
}

function main() {

  # REQUIRED
  local WAVEFRONT_TOKEN=
  local WAVEFRONT_URL="https:\/\/nimba.wavefront.com"
  local WF_CLUSTER=nimba
  local VERSION=$(cat ${REPO_ROOT}/release/OPERATOR_VERSION)
  local COLLECTOR_VERSION=$(cat ${REPO_ROOT}/release/COLLECTOR_VERSION)
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

  run_test "advanced-proxy"

  run_test "advanced-collector"

  run_test "advanced-default-config"

  run_test "basic" true
}

main "$@"
