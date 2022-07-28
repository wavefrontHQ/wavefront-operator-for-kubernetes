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
  echo "Waiting for all Pods to be 'Ready'"
  while ! kubectl wait --for=condition=Ready pod --all -l exclude-me!=true --all-namespaces &> /dev/null; do
    echo "Waiting for all Pods to be 'Ready'"
    sleep 5
  done
  echo "All Pods are Ready"
}

function create_cluster_name() {
  local K8S_ENV=$(cd ${REPO_ROOT}/hack/test && ./get-k8s-cluster-env.sh)
  echo $(whoami)-${K8S_ENV}-operator-$(date +"%y%m%d")
}

function check_arg() {
  usage=$1
  arg_name=$2
  arg_val=$3
  if [ -z "${arg_val}" ]; then
    echo "missing argument '${arg_name}'; usage '${usage}'"
    exit 1
  fi
}

function confirm() {
  if [ "${NON_INTERACTIVE}" != "" ]; then
    return 0
  fi

  message=$1
  read -p "${message}"' [y/n]: ' -n 1 -r
  echo

  # this creates output, which is basically like a return value in bash. Ugh, I'm sorry.
  [[ "${REPLY}" =~ ^[Yy]$ ]]

  # Keeping this for posterity.
  # I thought it would be cool for this function to run the command
  # but now I realize it's more normal for this to just return true/false
  # and let the script writer deal with that.
  # cmd_with_args=${@:2}
}
