function green() {
  echo -e $'\e[32m'$1$'\e[0m'
}

function red() {
  echo -e $'\e[31m'$1$'\e[0m'
}

function yellow() {
  echo -e $'\e[1;33m'$1$'\e[0m'
}

function print_msg_and_exit() {
  red "$1"
  exit 1
}

function pushd_check() {
  local d=$1
  pushd ${d} || print_msg_and_exit "Entering directory '${d}' with 'pushd' failed!"
}

function popd_check() {
  local d=$1
  popd || print_msg_and_exit "Leaving '${d}' with 'popd' failed!"
}

function wait_for_cluster_ready() {
  printf "Waiting for all Pods to be 'Ready' ..."
  while ! kubectl wait --for=condition=Ready pod --all -l exclude-me!=true --all-namespaces --timeout=5s &> /dev/null; do
    printf "."
    sleep 1
  done
  echo " done."
}

function create_cluster_name() {
  local K8S_ENV=$(cd ${REPO_ROOT}/hack/test && ./get-k8s-cluster-env.sh)
  echo $(whoami)-${K8S_ENV}-operator-$(date +"%y%m%d")
}