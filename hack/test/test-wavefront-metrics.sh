#!/bin/bash -e

REPO_ROOT=$(git rev-parse --show-toplevel)
source ${REPO_ROOT}/hack/test/k8s-utils.sh

function curl_query_to_wf_dashboard() {
  local query=$1
  # NOTE: any output inside this function is concatenated and used as the return value;
  # otherwise we would love to put a log such as this in here to give us more information:
  # echo "=============== Querying '$WF_CLUSTER' for query '${query}'"
  curl -s -X GET --header "Accept: application/json" \
    --header "Authorization: Bearer $WAVEFRONT_TOKEN" \
    "https://$WF_CLUSTER.wavefront.com/api/v2/chart/api?q=${query}&queryType=WQL&s=$AFTER_UNIX_TS&g=s&view=METRIC&sorted=false&cached=true&useRawQK=false" |
    jq '.timeseries[0].data[0][1]'
}

function wait_for_query_match_tags() {
  local query=$1
  local expected_tags_json=$2
  local actual_tags_json=$(mktemp)
  local loop_count=0

  printf "Querying for tags %s ..."  "$query"

  while [[ $loop_count -lt $MAX_QUERY_TIMES ]]; do
    loop_count=$((loop_count + 1))
    END_TIME="$(date '+%s')000"
    START_TIME="$(echo "`date +%s` - 120"| bc)000"
    curl -s -X GET --header "Accept: application/json" \
       --header "Authorization: Bearer $WAVEFRONT_TOKEN" \
       "https://$WF_CLUSTER.wavefront.com/api/v2/chart/api?q=${query}&queryType=WQL&s=$START_TIME&e=$END_TIME&g=m&i=false&strict=true&view=METRIC&includeObsoleteMetrics=false&sorted=false&cached=true&useRawQK=false" | \
       jq -S '.timeseries[0].tags' | \
       sort | sed 's/,//g' > "$actual_tags_json"
    printf "."
    if [ "$(comm -23 "$expected_tags_json" "$actual_tags_json")" == "" ]; then
      echo " done."
      return 0
    fi
    sleep $CURL_WAIT
  done

  if [ "$(comm -23 "$expected_tags_json" "$actual_tags_json")" != "" ]; then
    printf "\nChecking if expected tags are a subset of actual tags for query %s failed after attempting %s times.\n" "$query" "$MAX_QUERY_TIMES"
    echo "Actual tags are:"
    cat "$actual_tags_json"
    echo "Expected tags are:"
    cat "$expected_tags_json"
  fi
  return 1
}

function wait_for_query_match_exact() {
  local query=$1
  local expected=$2
  local actual
  local loop_count=0

  printf "Querying for exact match %s ..."  "$query"

  while [[ $loop_count -lt $MAX_QUERY_TIMES ]]; do
    loop_count=$((loop_count + 1))
    actual=$(curl_query_to_wf_dashboard "${query}")
    printf "."
    if echo "$actual $expected" | awk '{exit ($1 > $2 || $1 < $2)}'; then
        echo " done."
        return 0
    fi

    sleep $CURL_WAIT
  done

  if [[ $actual != $expected ]]; then
    echo "Checking wavefront dashboard metrics for $query failed after attempting $MAX_QUERY_TIMES times."
    echo "Actual is '$actual'"
    echo "Expected is '$expected'"
  fi
  return 1
}

function wait_for_query_non_zero() {
  local query_non_zero=$1
  local actual=0
  local loop_count=0

  printf "Querying for non zero %s ..."  "$query_non_zero"

  while true; do
    loop_count=$((loop_count + 1))
    actual=$(curl_query_to_wf_dashboard "${query_non_zero}")
    printf "."
    if [[ $actual != null && $actual != 0 ]] || [[ $loop_count -eq $MAX_QUERY_TIMES ]]; then
      break
    fi
    sleep $CURL_WAIT
  done
  echo " done."

  if [[ $actual == null || $actual == 0 ]]; then
    echo "Checking wavefront dashboard metrics for $query_non_zero failed after attempting $MAX_QUERY_TIMES times."
    echo "Actual is '$actual'"
    echo "Expected non zero"
    return 1
  fi
}

function print_usage_and_exit() {
  echo "Failure: $1"
  echo "Usage: $0 [flags] [options]"
  echo -e "\t-t wavefront token (required)"
  echo -e "\t-n config cluster name for metric grouping (required)"
  echo -e "\t-w wavefront instance name (default: 'nimba')"
  echo -e "\t-c collector version (default: load from 'release/COLLECTOR_VERSION')"
  echo -e "\t-o operator version (default: load from 'release/OPERATOR_VERSION')"
  echo -e "\t-e name of a file containing any extra asserts that should be made as part of this test (optional)"
  echo -e "\t-l name of test proxy used for logging (optional)"
  exit 1
}

function exit_on_fail() {
  # shellcheck disable=SC2068
  $@ # run all arguments as a command
  local exit_code=$?
  if [[ $exit_code != 0 ]]; then
    # shellcheck disable=SC2145
    echo "Command '$@' exited with exit code '$exit_code'"
    exit $exit_code
  fi
}

function main() {
  cd "$(dirname "$0")" # hack/test

  local AFTER_UNIX_TS="$(date '+%s')000"
  local MAX_QUERY_TIMES=90
  local CURL_WAIT=5

  # REQUIRED
  local WAVEFRONT_TOKEN=
  local CONFIG_CLUSTER_NAME=

  local EXPECTED_COLLECTOR_VERSION=$(cat ${REPO_ROOT}/release/COLLECTOR_VERSION)
  local EXPECTED_OPERATOR_VERSION=$(cat ${REPO_ROOT}/release/OPERATOR_VERSION)
  local WF_CLUSTER=nimba
  local EXTRA_TESTS=
  local LOGGING_TEST_PROXY_NAME=


  while getopts ":c:t:n:o:c:e:l:" opt; do
    case $opt in
    w)
      WF_CLUSTER="$OPTARG"
      ;;
    t)
      WAVEFRONT_TOKEN="$OPTARG"
      ;;
    n)
      CONFIG_CLUSTER_NAME="$OPTARG"
      ;;
    o)
      EXPECTED_OPERATOR_VERSION="$OPTARG"
      ;;
    c)
      EXPECTED_COLLECTOR_VERSION="$OPTARG"
      ;;
    e)
      EXTRA_TESTS="$OPTARG"
      ;;
    l)
      LOGGING_TEST_PROXY_NAME="$OPTARG"
      ;;
    \?)
      print_usage_and_exit "Invalid option: -$OPTARG"
      ;;
    esac
  done

  if [[ -z ${WAVEFRONT_TOKEN} ]]; then
    print_usage_and_exit "wavefront token required"
  fi
  if [[ -z ${CONFIG_CLUSTER_NAME} ]]; then
    print_usage_and_exit "config cluster name required"
  fi

  local COLLECTOR_VERSION_IN_DECIMAL="${EXPECTED_COLLECTOR_VERSION%.*}"
  local COLLECTOR_VERSION_IN_DECIMAL+="$(echo "${EXPECTED_COLLECTOR_VERSION}" | cut -d '.' -f3)"
  local COLLECTOR_VERSION_IN_DECIMAL="$(echo "${COLLECTOR_VERSION_IN_DECIMAL}" | sed 's/0$//')"

  wait_for_cluster_ready $NS

  local EXPECTED_TAGS_JSON=$(mktemp)
  jq -S -n --arg status Healthy \
     --arg proxy Healthy \
     --arg metrics Healthy \
     --arg logging Healthy \
     --arg version "$EXPECTED_OPERATOR_VERSION" \
     '$ARGS.named' | \
     sort | sed 's/,//g' > "$EXPECTED_TAGS_JSON"

  exit_on_fail wait_for_query_match_tags "at(%22end%22%2C%202m%2C%20ts(%22kubernetes.observability.status%22%2C%20cluster%3D%22${CONFIG_CLUSTER_NAME}%22))" "${EXPECTED_TAGS_JSON}"
  exit_on_fail wait_for_query_match_exact "ts(kubernetes.collector.version%2C%20cluster%3D%22${CONFIG_CLUSTER_NAME}%22%20AND%20installation_method%3D%22operator%22)" "${COLLECTOR_VERSION_IN_DECIMAL}"
  exit_on_fail wait_for_query_non_zero "ts(kubernetes.cluster.pod.count%2C%20cluster%3D%22${CONFIG_CLUSTER_NAME}%22)"

  if [[ ! -z ${LOGGING_TEST_PROXY_NAME} ]]; then
    exit_on_fail wait_for_query_non_zero "ts(~proxy.logs.*.received.bytes%2C%20source%3D%22${LOGGING_TEST_PROXY_NAME}%22)"
  fi

  if [[ -f "${EXTRA_TESTS}" ]]; then
    source "${EXTRA_TESTS}"
  else
    echo "no extra tests"
  fi
}

main "$@"
