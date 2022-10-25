#!/usr/bin/env bash
set -e

cd "$(dirname "$0")"

function component-image-refs() {
  local name="$1"
  delimiter='\}\}/'
  (
    while IFS= read -r line; do
      echo "${line#*$delimiter}"
    done < <(grep "/${name}:" ../deploy/internal/**/*.yaml | uniq)
  ) | uniq
}

component-image-refs "kubernetes-collector"
component-image-refs "kubernetes-operator-fluentd"
component-image-refs "proxy"
