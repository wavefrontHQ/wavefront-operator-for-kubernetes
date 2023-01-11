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

  sed "s/YOUR_CLUSTER_NAME/$cluster_name/g" ${REPO_ROOT}/hack/test/deploy/scenarios/wavefront-$type.yaml |
    sed "s/YOUR_WAVEFRONT_URL/$wf_url/g" |
    sed "s/YOUR_API_TOKEN/${WAVEFRONT_TOKEN}/g" |
    sed "s/YOUR_NAMESPACE/${NS}/g" >hack/test/_v1alpha1_wavefront_test.yaml

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

  proxyLogErrorCount=$(kubectl logs deployment/wavefront-proxy -n $NS | grep " ERROR " | wc -l | sed -e 's/^[[:space:]]*//' -e 's/[[:space:]]*$//')
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

function checks_to_remove() {
  local file_name=$1
  local component_name=$2
  local checks=$3
  local tempFile

  for i in ${checks//,/ }; do
    tempFile=$(mktemp)
    local excludeCheck=$(echo $i | sed -r 's/:/ /g')
    local awk_command="!/.*$excludeCheck.*$component_name|.*$component_name.*$excludeCheck/"
    cat "$file_name" | awk "$awk_command" >"$tempFile" && mv "$tempFile" "$file_name"
  done
}

function run_static_analysis() {
  local type=$1
  local k8s_env=$(k8s_env)
  echo "Running static analysis ..."

  local resources_yaml_file=$(mktemp)
  local exit_status=0
  kubectl get "$(kubectl api-resources --verbs=list --namespaced -o name | tr '\n' ',' | sed s/,\$//)" --ignore-not-found -n $NS -o yaml |
    yq '.items[] | split_doc' - >"$resources_yaml_file"

  echo "Running static analysis: kube-linter"

  local kube_lint_results_file=$(mktemp)
  local kube_lint_check_errors=$(mktemp)
  ${REPO_ROOT}/bin/kube-linter lint "$resources_yaml_file" --format json 1>"$kube_lint_results_file" 2>/dev/null || true

  local current_lint_errors="$(jq '.Reports | length' "$kube_lint_results_file")"
  yellow "Kube linter error count: ${current_lint_errors}"

  jq -r '.Reports[] | "|" + .Check + "|  " +.Object.K8sObject.GroupVersionKind.Kind + " " + .Object.K8sObject.Namespace + "/" +  .Object.K8sObject.Name + ": " + .Diagnostic.Message' "$kube_lint_results_file" 1>"$kube_lint_check_errors" 2>/dev/null || true

  #REMOVE KNOWN CHECKS
  #non root checks for logging
  checks_to_remove "$kube_lint_check_errors" "wavefront-logging" "run-as-non-root,no-read-only-root-fs"
  #sensitive-host-mounts checks for the collector
  checks_to_remove "$kube_lint_check_errors" "collector" "sensitive-host-mounts"

  current_lint_errors=$(cat "$kube_lint_check_errors" | wc -l)
  yellow "Kube linter error count (with known errors removed): ${current_lint_errors}"
  local known_lint_errors=0
  if [ $current_lint_errors -gt $known_lint_errors ]; then
    red "Failure: Expected error count = $known_lint_errors"
    cat "$kube_lint_check_errors"
    exit_status=1
  fi

  echo "Running static analysis: kube-score"
  local kube_score_results_file=$(mktemp)
  local kube_score_critical_errors=$(mktemp)
  ${REPO_ROOT}/bin/kube-score score "$resources_yaml_file" --ignore-test pod-networkpolicy --output-format ci >"$kube_score_results_file" || true

  grep '\[CRITICAL\]' "$kube_score_results_file" >"$kube_score_critical_errors"
  local current_score_errors=$(cat "$kube_score_critical_errors" | wc -l)
  yellow "Kube score error count: ${current_score_errors}"

  #REMOVE KNOWN CHECKS
  #non root checks for logging
  checks_to_remove "$kube_score_critical_errors" "wavefront-logging" "security:context,low:user:ID,low:group:ID"
  if [[ "$k8s_env" == "Kind" ]]; then
    checks_to_remove "$kube_score_critical_errors" "wavefront-controller-manager" "ImagePullPolicy"
  fi

  current_score_errors=$(cat "$kube_score_critical_errors" | wc -l)
  yellow "Kube score error count (with known errors removed): ${current_score_errors}"
  local known_score_errors=0
  if [ $current_score_errors -gt $known_score_errors ]; then
    red "Failure: Expected error count = $known_score_errors"
    cat "$kube_score_critical_errors"
    exit_status=1
  fi

  echo "Running static analysis: ServiceAccount automountServiceAccountToken checks"
  local automountToken=
  local service_accounts=$(kubectl get serviceaccounts -l app.kubernetes.io/name=wavefront -n $NS -o name | tr '\n' ',' | sed "s/serviceaccount\///g" | sed s/,\$//)

  for i in ${service_accounts//,/ }; do
    automountToken=$(kubectl get serviceaccount $i -n $NS -o=jsonpath='{.automountServiceAccountToken}' | tr -d '\n')
    if [[ $automountToken != "false" ]]; then
      red "Failure: Expected automountToken in $i to be \"false\", but was $automountToken"
      exit 1
    fi
  done

  echo "Running static analysis: Pod automountServiceAccountToken checks"
  local pods=$(kubectl get pods -l app.kubernetes.io/name=wavefront -n $NS -o name | tr '\n' ',' | sed "s/pod\///g" | sed s/,\$//)

  for i in ${pods//,/ }; do
    automountToken=$(kubectl get pod $i -n $NS -o=jsonpath='{.spec.automountServiceAccountToken}' | tr -d '\n')
    if [[ $automountToken == "" ]]; then
      red "Failure: Expected automountToken in $i to be set"
      exit 1
    fi
  done

  if [[ $exit_status -ne 0 ]]; then
    exit $exit_status
  fi
}

function run_logging_checks() {
  printf "Running logging checks ..."
  local max_logs_received=0
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

function run_logging_integration_checks() {
  printf "Running logging checks with test-proxy ..."

  # send request to the fake proxy control endpoint and check status code for success
  kill $(jobs -p) &>/dev/null || true
  sleep 3
  kubectl --namespace "$NS" port-forward deploy/test-proxy 8888 &
  trap 'kill $(jobs -p) &>/dev/null || true' EXIT
  sleep 3

  RES=$(mktemp)

  for _ in {1..10}; do
    RES_CODE=$(curl --silent --output "$RES" --write-out "%{http_code}" "http://localhost:8888/logs/assert")
    if [[ $RES_CODE -eq 200 ]]; then
      break
    fi
    sleep 1
  done

  # Helpful for debugging:
  # cat "${RES}" >/tmp/test

  if [[ $RES_CODE -eq 204 ]]; then
    red "Logs were never received by test proxy"
    kubectl -n observability-system exec deployment/test-proxy -- cat /logs/test-proxy.log
    exit 1
  fi

  # TODO look at result and pass or fail test
  if [[ $RES_CODE -gt 399 ]]; then
    red "LOGGING ASSERTION FAILURE"
    kubectl -n observability-system exec deployment/test-proxy -- cat /logs/test-proxy.log
    exit 1
  fi

  hasValidFormat=$(jq -r .hasValidFormat "${RES}")
  if [[ ${hasValidFormat} -ne 1 ]]; then
    red "Test proxy received logs with invalid format"
    kubectl -n observability-system exec deployment/test-proxy -- cat /logs/test-proxy.log
    exit 1
  fi

  hasValidTags=$(jq -r .hasValidTags "${RES}")
  missingExpectedTags="$(jq .missingExpectedTags "${RES}")"
  missingExpectedTagsCount="$(jq .missingExpectedTagsCount "${RES}")"

  emptyExpectedTags="$(jq .emptyExpectedTags "${RES}")"
  emptyExpectedTagsCount="$(jq .emptyExpectedTagsCount "${RES}")"

  unexpectedAllowedLogs="$(jq .unexpectedAllowedLogs "${RES}")"
  unexpectedAllowedLogsCount="$(jq .unexpectedAllowedLogsCount "${RES}")"

  unexpectedDeniedTags="$(jq .unexpectedDeniedTags "${RES}")"
  unexpectedDeniedTagsCount="$(jq .unexpectedDeniedTagsCount "${RES}")"

  receivedLogCount=$(jq .receivedLogCount "${RES}")

  if [[ ${hasValidTags} -ne 1 ]]; then
    red "Invalid tags were found:"
    if [[ ${missingExpectedTags} != "null" ]]; then
      echo ""
      red "* Test proxy received logs (${missingExpectedTagsCount}/${receivedLogCount} logs) that were missing expected tags:"
      red "${missingExpectedTags}"
    fi

    if [[ ${emptyExpectedTags} != "null" ]]; then
      echo ""
      red "* Test proxy received logs (${emptyExpectedTagsCount}/${receivedLogCount} logs) with expected tags that were empty:"
      red "${emptyExpectedTags}"
    fi

    if [[ ${unexpectedAllowedLogs} != "null" ]]; then
      echo ""
      red "* Test proxy received (${unexpectedAllowedLogsCount}/${receivedLogCount} logs) logs that should not have been there because none of their tags were in the allowlist:"
      red "${unexpectedAllowedLogs}"
    fi

    if [[ ${unexpectedDeniedTags} != "null" ]]; then
      echo ""
      red "* Test proxy received (${unexpectedDeniedTagsCount}/${receivedLogCount} logs) logs that should not have been there because some of their tags were in the denylist:"
      red "${unexpectedDeniedTags}"
    fi

    exit 1
  fi

  echo "Integration test complete. ${receivedLogCount} logs were checked."
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

  if [[ " ${checks[*]} " =~ " logging-integration-checks " ]]; then
    run_logging_integration_checks
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
      "logging-integration"
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
  if [[ " ${tests_to_run[*]} " =~ " logging-integration " ]]; then
    run_test "logging-integration" "logging-integration-checks"
  fi
}

main "$@"
