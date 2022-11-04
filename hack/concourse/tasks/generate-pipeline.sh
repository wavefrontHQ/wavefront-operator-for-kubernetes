#!/usr/bin/env bash
set -e

function get_feature_branches() {
    git ls-remote --heads \
      | grep -E 'refs/heads/K8SSAAS' \
      | grep -oE 'K8SSAAS-\d{3,4}.*$'
}

function get_resources() {
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
resources=$(get_resources $(cat feature-branches.txt))

awk -v from='#+ {{ feature_branch_resources }}'\
 to=$resources \
 '{gsub(from,to)}' hack/concourse/pipeline.yaml

