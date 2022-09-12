#!/bin/bash -e
REPO_ROOT=$(git rev-parse --show-toplevel)

cd ${REPO_ROOT}

git config --global user.email "svc.wf-jenkins@vmware.com"
git config --global user.name "svc.wf-jenkins"
git remote set-url origin https://${TOKEN}@github.com/wavefronthq/wavefront-operator-for-kubernetes.git

RELEASE_VERSION=$(cat ./release/OPERATOR_VERSION)
NEW_VERSION=$(semver-cli inc patch "$RELEASE_VERSION")
VERSION=$NEW_VERSION$VERSION_POSTFIX make generate-kubernetes-yaml

cp deploy/kubernetes/wavefront-operator.yaml build/wavefront-operator.yaml
git fetch
git checkout .
git checkout $GIT_BRANCH || git checkout -b $GIT_BRANCH

ls | grep -v build | xargs rm -rf
mv build/wavefront-operator.yaml wavefront-operator$VERSION_POSTFIX.yaml

git add wavefront-operator$VERSION_POSTFIX.yaml
git commit -m "add CRD"
git push --set-upstream origin $GIT_BRANCH

PR_URL=$(curl \
  -X POST \
  -H "Authorization: token ${TOKEN}" \
  -d "{\"head\":\"${GIT_BRANCH}\",\"base\":\"rc\",\"title\":\"Add release candidate rc${GIT_BRANCH}\"}" \
  https://api.github.com/repos/wavefrontHQ/wavefront-operator-for-kubernetes/pulls |
  jq -r '.html_url')

PULL_NUMBER=$(echo ${PR_URL} | sed 's:.*/::')

MERGE_PR_URL=$(curl \
  -X PUT \
  -H "Accept: application/vnd.github+json" \
  -H "Authorization: token ${TOKEN}" \
  https://api.github.com/repos/OWNER/REPO/pulls/${PULL_NUMBER}/merge)

echo "PR URL: ${PR_URL}"