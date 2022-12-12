#!/bin/bash -e

REPO_ROOT=$(git rev-parse --show-toplevel)
source ${REPO_ROOT}/hack/test/k8s-utils.sh
NS=observability-system

function print_usage_and_exit() {
  echo "Failure: $1"
  echo "Usage: $0 [flags] [options]"
  echo -e "\t-t wavefront token (required)"
  echo -e "\t-c wavefront instance name (default: 'nimba')"
  echo -e "\t-v operator version (default: load from 'release/OPERATOR_VERSION')"
  echo -e "\t-n config cluster name for metric grouping (default: \$(whoami)-<default version from file>-release-test)"
  echo -e "\t-d namespace to create CR in (default: observability-system"
  echo -e "\t-r tests to run (runs all by default)"
  exit 1
}

function setup_test() {
  local type=$1
  local wf_url="${2:-${WAVEFRONT_URL}}"
  local cluster_name=${CONFIG_CLUSTER_NAME}-$type

  echo "Deploying Wavefront CR with Cluster Name: $cluster_name ..."

  wait_for_cluster_ready "$NS"

  sed "s/YOUR_CLUSTER_NAME/$cluster_name/g"  ${REPO_ROOT}/hack/test/deploy/scenarios/wavefront-$type.yaml  |
   sed "s/YOUR_WAVEFRONT_URL/$wf_url/g" |
   sed "s/YOUR_API_TOKEN/${WAVEFRONT_TOKEN}/g" |
    sed "s/YOUR_NAMESPACE/${NS}/g" > hack/test/_v1alpha1_wavefront_test.yaml

  kubectl apply -f hack/test/_v1alpha1_wavefront_test.yaml

  wait_for_cluster_ready "$NS"
}

function run_test_wavefront_metrics() {
  local type=$1
  local cluster_name=${CONFIG_CLUSTER_NAME}-$type
  echo "Running test wavefront metrics, cluster_name $cluster_name ..."

  ${REPO_ROOT}/hack/test/test-wavefront-metrics.sh -t ${WAVEFRONT_TOKEN} -n $cluster_name -e "$type-test.sh" -o ${VERSION}
}

function run_health_checks() {
  local type=$1
  local should_be_healthy="${2:-true}"
  printf "Running health checks ..."

  local health_status=
  for _ in {1..120}; do
    health_status=$(kubectl get wavefront -n $NS --request-timeout=10s -o=jsonpath='{.items[0].status.status}') || true
    if [[ "$health_status" == "Healthy" ]]; then
      break
    fi
    printf "."
    sleep 2
  done

  if [[ "$health_status" != "Healthy" ]]; then
    red "Health status for $type: expected = true, actual = $health_status"
    exit 1
  fi

  proxyLogErrorCount=$(kubectl logs deployment/wavefront-proxy -n $NS | grep " ERROR "| wc -l | sed -e 's/^[[:space:]]*//' -e 's/[[:space:]]*$//')
  if [[ $proxyLogErrorCount -gt 0 ]]; then
    red "Expected proxy log error count of 0, but got $proxyLogErrorCount"
    exit 1
  fi

  echo " done."
}

function run_unhealthy_checks() {
  local type=$1
  echo "Running unhealthy checks ..."

  for _ in {1..10}; do
    health_status=$(kubectl get wavefront -n $NS --request-timeout=10s -o=jsonpath='{.items[0].status.status}') || true
    if [[ "$health_status" == "Unhealthy" ]]; then
      break
    fi
    printf "."
    sleep 1
  done

  if [[ "$health_status" != "Unhealthy" ]]; then
    red "Health status for $type: expected = false, actual = $health_status"
    exit 1
  else
    yellow "Success got expected error: $(kubectl get wavefront -n $NS -o=jsonpath='{.items[0].status.message}')"
  fi
}

function clean_up_test() {
  local type=$1
  echo "Cleaning Up ..."

  kubectl delete -f hack/test/_v1alpha1_wavefront_test.yaml --timeout=10s

  wait_for_proxy_termination "$NS"
}

function run_static_analysis() {
  local type=$1
  echo "Running static analysis ..."

  local resources_yaml_file=$(mktemp)
  local exit_status=0
  kubectl get "$(kubectl api-resources --verbs=list --namespaced -o name | tr '\n' ',' | sed s/,\$//)" --ignore-not-found -n $NS -o yaml \
  | yq '.items[] | split_doc' - > "$resources_yaml_file"

  # Ideally we want to fail when a non-zero error count is identified. Until we get to a zero error count, use the known
  # error count to pass.
  echo "Running static analysis: kube-linter"
  local kube_lint_results_file=$(mktemp)
  ${REPO_ROOT}/bin/kube-linter lint "$resources_yaml_file" --format json 1> "$kube_lint_results_file" 2>/dev/null || true

  local current_lint_errors="$(jq '.Reports | length' "$kube_lint_results_file")"
  yellow "Kube linter error count: ${current_lint_errors}"
  local known_lint_errors=8
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
  local known_score_errors=11
  if [ $current_score_errors -gt $known_score_errors ]; then
    red "Failure: Expected error count = $known_score_errors"
    grep '\[CRITICAL\]' "$kube_score_results_file"
    exit_status=1
  fi

  if [[ $exit_status -ne 0 ]]; then
    exit $exit_status
  fi
}

function run_logging_checks() {
  printf "Running logging checks ..."
  local max_logs_received=0;
  for _ in {1..12}; do
    max_logs_received=$(kubectl -n $NS logs -l app.kubernetes.io/name=wavefront -l app.kubernetes.io/component=proxy --tail=-1 | grep "Logs received" | awk 'match($0, /[0-9]+ logs\/s/) { print substr( $0, RSTART, RLENGTH )}' | awk '{print $1}' | sort -n | tail -n1 2>/dev/null)
    if [[ $max_logs_received -gt 0 ]]; then
      break
    fi
    sleep 5
  done

  if [[ $max_logs_received -eq 0 ]]; then
    red "Expected max logs received to be greater than 0, but got $max_logs_received"
    exit 1
  fi
  echo " done."
}

function run_logging_checks_test_proxy() {
  printf "Running logging checks with test-proxy ..."

  FAKE_NS=observability-fake-proxy
  # we have a running cluster with operator deployed

  # deploy fake proxy into cluster in separate namespace
  kubectl create namespace "$FAKE_NS"
  kubectl apply -f "${REPO_ROOT}/hack/test/test-proxy.yaml"

  # edit the fluentd config to overwrite the config to point to the fake proxy
    # TODO replace 'wavefront-proxy:2878' with 'test-proxy.observability-fake-proxy.svc.cluster.local:9999'
  # apply fluentd config back to cluster and wait for it to reload


  # send request to the fake proxy control endpoint and check status code for success
  kill $(jobs -p) &>/dev/null || true
  kubectl --namespace "$FAKE_NS" port-forward deploy/test-proxy 8888 &
  trap 'kill $(jobs -p) &>/dev/null || true' EXIT

  RES_CODE=$(curl --silent --write-out "%{http_code}" --data "expected_format=json_array" "http://localhost:8888/logs/assert")

  if [[ $RES_CODE -gt 399 ]]; then
    red "INVALID METRICS"
    jq -r '.[]' "${RES}"
    exit 1
  fi

  # delete fake proxy and reset logging config

  echo " done."
}

function run_test() {
  local type=$1
  shift
  local checks=("$@")
  echo ""
  green "Running test $type"

  setup_test $type

  if [[ " ${checks[*]} " =~ " unhealthy " ]]; then
    run_unhealthy_checks $type
  elif [[ " ${checks[*]} " =~ " health " ]]; then
    run_health_checks $type
  fi

  if [[ " ${checks[*]} " =~ " static_analysis " ]]; then
    run_static_analysis $type
  fi

  if [[ " ${checks[*]} " =~ " test_wavefront_metrics " ]]; then
    run_test_wavefront_metrics $type
  fi

  if [[ " ${checks[*]} " =~ " logging " ]]; then
    run_logging_checks
  fi

  if [[ " ${checks[*]} " =~ " logging-test-proxy " ]]; then
    run_logging_checks_test_proxy
  fi

  clean_up_test $type
  green "Successfully ran $type test!"
}

function main() {

  # REQUIRED
  local WAVEFRONT_TOKEN=

  local WAVEFRONT_URL="https:\/\/nimba.wavefront.com"
  local WF_CLUSTER=nimba
  local VERSION=$(cat ${REPO_ROOT}/release/OPERATOR_VERSION)
  local K8S_ENV=$(cd ${REPO_ROOT}/hack/test && ./get-k8s-cluster-env.sh)
  local CONFIG_CLUSTER_NAME=$(create_cluster_name)
  local tests_to_run=()

  while getopts ":c:t:v:n:d:r:" opt; do
    case $opt in
    t)
      WAVEFRONT_TOKEN="$OPTARG"
      ;;
    c)
      WF_CLUSTER="$OPTARG"
      ;;
    v)
      VERSION="$OPTARG"
      ;;
    n)
      CONFIG_CLUSTER_NAME="$OPTARG"
      ;;
    r)
      tests_to_run+=("$OPTARG")
      ;;
    d)
      NS="$OPTARG"
      ;;
    \?)
      print_usage_and_exit "Invalid option: -$OPTARG"
      ;;
    esac
  done

  if [[ ${#tests_to_run[@]} -eq 0 ]]; then
    tests_to_run=(
      "validation-errors"
      "validation-legacy"
      "allow-legacy-install"
      "basic"
      "advanced"
    )
  fi

  if [[ -z ${WAVEFRONT_TOKEN} ]]; then
    print_usage_and_exit "wavefront token required"
  fi

  if [[ -z ${CONFIG_CLUSTER_NAME} ]]; then
    CONFIG_CLUSTER_NAME=$(create_cluster_name)
  fi

  cd "$REPO_ROOT"

  if [[ " ${tests_to_run[*]} " =~ " validation-errors " ]]; then
    run_test "validation-errors" "unhealthy"
  fi
  if [[ " ${tests_to_run[*]} " =~ " validation-legacy " ]]; then
    run_test "validation-legacy" "unhealthy"
  fi
  if [[ " ${tests_to_run[*]} " =~ " allow-legacy-install " ]]; then
    run_test "allow-legacy-install" "healthy"
  fi
  if [[ " ${tests_to_run[*]} " =~ " basic " ]]; then
    run_test "basic" "health" "static_analysis"
  fi
  if [[ " ${tests_to_run[*]} " =~ " advanced " ]]; then
    run_test "advanced" "health" "test_wavefront_metrics" "logging"
  fi
  if [[ " ${tests_to_run[*]} " =~ " logging " ]]; then
    run_test "logging-test-proxy"
  fi
}

main "$@"
