#!/usr/bin/env bash
set -e

function get_resources() {
  echo "#@data/values"
  echo '#@ load("@ytt:overlay", "overlay")'
  echo "---"
  echo "resources:"
  echo "#@overlay/append"
  for feature_branch in "${@}" ; do
    jira=$(echo $feature_branch | grep -oE 'K8SSAAS-\d{3,4}')
    cat <<- EOD
- name: wavefront-operator-${jira}
  type: git
  source:
    uri: git@github.com:wavefrontHQ/wavefront-operator-for-kubernetes.git
    branch: ${feature_branch}
    private_key: ((osspi.jcornish-github-private-key))
EOD
  done
}

function ensure_ytt() {
  if ! command -v ytt; then
    curl https://github.com/vmware-tanzu/carvel-ytt/releases/download/v0.43.0/ytt-linux-amd64 \
      --output /usr/local/bin/ytt
    chmod +x /usr/local/bin/ytt
    ytt version
  fi
}

ensure_ytt

echo "Generating pipeline"
get_resources $(cat feature_branches/list.txt) > ci_repo/hack/concourse/yamlbits/feature_branch_resources.yaml

ytt -f ci_repo/hack/concourse/yamlbits \
  > pipeline.yaml

cat pipeline.yaml
