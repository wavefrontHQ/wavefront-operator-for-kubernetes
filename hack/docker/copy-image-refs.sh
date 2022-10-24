#!/usr/bin/env bash
set -e

REPO_ROOT=$(git rev-parse --show-toplevel)
source "${REPO_ROOT}/hack/test/k8s-utils.sh"
cd "${REPO_ROOT}"

function print_usage_and_exit() {
  echo "Failure: $1"
  echo "Usage: $0 [flags] [options]"
  echo "expects one image ref per line on stdin"
  echo -e "\t-s source image prefix"
  echo -e "\t-d destination image prefix"
  exit 1
}

function copy-image-ref() {
    local image_ref="$1"
    local src_prefix="$2"
    local dst_prefix="$3"
    docker buildx imagetools create -t "$dst_prefix/$image_ref" "$src_prefix/$image_ref"
}

function main() {
  local src_prefix=
  local dst_prefix=

  while getopts ":s:d:" opt; do
    case $opt in
    s)
      src_prefix="$OPTARG"
      ;;
    d)
      dst_prefix="$OPTARG"
      ;;
    \?)
      print_usage_and_exit "Invalid option: -$OPTARG"
      ;;
    esac
  done

  if [[ -z "$src_prefix" ]]; then
    print_usage_and_exit "-s required"
  fi

  if [[ -z "$dst_prefix" ]]; then
    print_usage_and_exit "-d required"
  fi

  while IFS='$\n' read -r image_ref; do
      copy-image-ref "$image_ref" "$src_prefix" "$dst_prefix"
  done
}

main "$@"
