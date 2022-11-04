#!/usr/bin/env bash
set -e

function get_feature_branches() {
    git ls-remote --heads \
      | grep -E 'refs/heads/K8SSAAS' \
      | grep -oE 'K8SSAAS-\d{3,4}.*$'
}

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

echo "Generating pipeline"
#get_feature_branches > feature-branches.txt
get_resources $(cat feature-branches.txt) > hack/concourse/yamlbits/feature_branch_resources.yaml

ytt -f hack/concourse/yamlbits \
  > pipeline.yaml
