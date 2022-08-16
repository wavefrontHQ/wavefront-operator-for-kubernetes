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
  local should_run_static_analysis="${2:-false}"
  local should_be_healthy="${3:-true}"
  local cluster_name=${CONFIG_CLUSTER_NAME}-$type
  local proxyLogErrorCount=0

  echo "Running $type CR"

  wait_for_cluster_ready

  sed "s/YOUR_CLUSTER_NAME/$cluster_name/g"  ${REPO_ROOT}/hack/test/deploy/scenarios/wavefront-$type.yaml  |
   sed "s/YOUR_WAVEFRONT_URL/${WAVEFRONT_URL}/g" > hack/test/_v1alpha1_wavefront_test.yaml

  kubectl apply -f hack/test/_v1alpha1_wavefront_test.yaml

  wait_for_cluster_ready

  if "$should_be_healthy"; then
    if "$should_run_static_analysis"; then
      run_static_analysis
    fi

    echo "Running test-wavefront-metrics"
    ${REPO_ROOT}/hack/test/test-wavefront-metrics.sh -t ${WAVEFRONT_TOKEN} -n $cluster_name -v ${COLLECTOR_VERSION} -e "$type-test.sh"

    health_status=$(kubectl get wavefront -n wavefront -o=jsonpath='{.items[0].status.status}')
    if [[ "$health_status" != "Healthy" ]]; then
      red "Health status for $type: expected = true, actual = $health_status"
      exit 1
    fi
    green "Success!"
  else
    sleep 1
    health_status=$(kubectl get wavefront -n wavefront -o=jsonpath='{.items[0].status.status}')
    if [[ "$health_status" != "Unhealthy" ]]; then
      red "Health status for $type: expected = false, actual = $health_status"
      exit 1
    else
      green "Success got expected error: $(kubectl get wavefront -n wavefront -o=jsonpath='{.items[0].status.message}')"
    fi
  fi

  proxyLogErrorCount=$(kubectl logs deployment/wavefront-proxy -n wavefront | grep error | wc -l | sed -e 's/^[[:space:]]*//' -e 's/[[:space:]]*$//')
  if [[ $proxyLogErrorCount > 0 ]]; then
    red "Expected proxy log error count of 0, but got $proxyLogErrorCount"
    exit 1
  fi

  kubectl delete -f hack/test/_v1alpha1_wavefront_test.yaml
}

function run_static_analysis() {
  local resources_yaml_file=$(mktemp)
  local exit_status=0
  kubectl get "$(kubectl api-resources --verbs=list --namespaced -o name | tr '\n' ',' | sed s/,\$//)" --ignore-not-found -n wavefront -o yaml \
  | yq '.items[] | split_doc' - > "$resources_yaml_file"

  # Ideally we want to fail when a non-zero error count is identified. Until we get to a zero error count, use the known
  # error count to pass.
  echo "Running static analysis: kube-linter"
  local kube_lint_results_file=$(mktemp)
  ${REPO_ROOT}/bin/kube-linter lint "$resources_yaml_file" --format json 1> "$kube_lint_results_file" 2>/dev/null || true

  local current_lint_errors="$(jq '.Reports | length' "$kube_lint_results_file")"
  yellow "Kube linter error count: ${current_lint_errors}"
  local known_lint_errors=10
  if [ $current_lint_errors -gt $known_lint_errors ]; then
    red "Failure: Expected error count = $known_lint_errors"
    jq -r '.Reports[] | .Object.K8sObject.GroupVersionKind.Kind + " " + .Object.K8sObject.Namespace + "/" +  .Object.K8sObject.Name + ": " + .Diagnostic.Message' "$kube_lint_results_file"
    exit_status=1
  fi

  echo "Running static analysis: kube-score"
  local kube_score_results_file=$(mktemp)
  ${REPO_ROOT}/bin/kube-score score "$resources_yaml_file" --output-format ci> "$kube_score_results_file" || true

  local current_score_errors=$(grep '\[CRITICAL\]' "$kube_score_results_file" | wc -l)
  yellow "Kube score error count: ${current_score_errors}"
  local known_score_errors=20
  if [ $current_score_errors -gt $known_score_errors ]; then
    red "Failure: Expected error count = $known_score_errors"
    grep '\[CRITICAL\]' "$kube_score_results_file"
    exit_status=1
  fi

  if [[ $exit_status -ne 0 ]]; then
    exit $exit_status
  fi
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

  run_test "validation-errors" false false
}

main "$@"
