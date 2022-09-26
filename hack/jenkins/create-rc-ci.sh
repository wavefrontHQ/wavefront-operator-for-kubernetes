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

git checkout $GIT_BRANCH

ls | grep -v build | xargs rm -rf
mv build/wavefront-operator.yaml wavefront-operator.yaml

git add --all .
git commit -m "updates wavefront-operator.yaml"
git push