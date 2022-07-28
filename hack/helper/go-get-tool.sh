#!/bin/bash -e

REPO_ROOT=$(git rev-parse --show-toplevel)
source ${REPO_ROOT}/hack/test/k8s-utils.sh
USAGE='k8po go-get-tool <tool_path> <tool_mod_link>'

function main() {
  tool_path=$1; tool_mod_link=$2;

  check_arg "${USAGE}" tool_path $tool_path; check_arg "${USAGE}" tool_mod_link $tool_mod_link;

  go_version=$(go version | awk -F ' ' '{print $3}')
  if [ "${go_version}" == 'go1.18' ]; then
    confirm 'go version is 1.18 and this will assuredly break things if you are in the Operator! Proceed?' || exit 0
  fi

  repo_root=$(git rev-parse --show-toplevel)
  tool_dir="${repo_root}/bin"
  if [ ! -d "${tool_dir}" ]; then
    echo "no bin folder found at ${tool_dir}; creating"
    mkdir -p "${tool_dir}"
  fi

  if [ -f "${tool_path}" ]; then
    echo "tool already found at path '${tool_path}'; exiting"
    exit 0
  fi

  TMP_DIR=$(mktemp -d)
  # GLOSSARY/WTF: push and pop output redirect
  # they are annoying to I redirect it to /dev/null.
  # another option is to create an alias for pushd and popd.
  pushd "${TMP_DIR}" > /dev/null
    go mod init tmp
    GOBIN="${tool_dir}" go install "${tool_mod_link}"
  popd > /dev/null
  rm -rf $TMP_DIR
}

main $@
