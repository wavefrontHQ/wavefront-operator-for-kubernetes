#!/usr/bin/env bash

function ensure_ytt() {
  if ! command -v wget; then
    apt update
    apt install -y wget
  fi

  if ! command -v ytt; then
    wget https://github.com/vmware-tanzu/carvel-ytt/releases/download/v0.43.0/ytt-linux-amd64
    chmod +x ytt-linux-amd64
    mv ytt-linux-amd64 /usr/local/bin/ytt
    ytt version
  fi
}

function get_jira_shortname() {
    echo $1 | grep -o 'K8SSAAS-[0-9]*' | awk '{print tolower($0)}'
}

function get_resources() {
  cat << EOF
#@data/values
#@ load("@ytt:overlay", "overlay")
---
resources:
#@overlay/append
EOF
  for feature_branch in "${@}" ; do
    jira=$(get_jira_shortname $feature_branch)
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

function get_jobs() {
  cat << EOF
#@data/values
#@ load("@ytt:overlay", "overlay")
---
jobs:
#@overlay/append
EOF
  for feature_branch in "${@}" ; do
    jira=$(get_jira_shortname $feature_branch)
    cat <<- EOD
- name: setup-cluster-${jira}
  plan:
    - get: wavefront-operator-ci
      passed: [update-self]
    - get: wavefront-operator-${jira}
      trigger: true
    - task: create-cluster-${jira}
      file: wavefront-operator-ci/hack/concourse/tasks/create-cluster.yaml
      input_mapping:
        ci_repo: wavefront-operator-ci
      params:
        CLUSTER_NAME: k8po-feature-cluster-${jira}
EOD
  done
}

ensure_ytt

echo "Generating pipeline"
get_resources $(cat feature_branches/list.txt) > ci_repo/hack/concourse/yamlbits/feature_branch_resources.yaml
get_jobs $(cat feature_branches/list.txt) > ci_repo/hack/concourse/yamlbits/feature_branch_jobs.yaml

ytt -f ci_repo/hack/concourse/yamlbits \
  > generated-pipeline/pipeline.yaml

echo "Generated Pipeline:"
cat generated-pipeline/pipeline.yaml
