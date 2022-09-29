#!/bin/bash -e
cd "$(git rev-parse --show-toplevel)"

git config --global user.email "svc.wf-jenkins@vmware.com"
git config --global user.name "svc.wf-jenkins"
#git remote set-url origin https://${TOKEN}@github.com/wavefronthq/wavefront-operator-for-kubernetes.git

RELEASE_VERSION=$(cat ./release/OPERATOR_VERSION)
NEW_VERSION=$(semver-cli inc patch "$RELEASE_VERSION")

VERSION=$NEW_VERSION$VERSION_POSTFIX make generate-kubernetes-yaml
cp deploy/kubernetes/wavefront-operator.yaml build/wavefront-operator.yaml

CURRENT_BRANCH=$(git rev-parse --abbrev-ref HEAD)

git fetch
git checkout rc
git reset --hard origin/rc

git clean -dfx -e build
mv build/wavefront-operator.yaml "wavefront-operator-${CURRENT_BRANCH}.yaml"

git add --all .
git commit -m "add CRD"
git push rc
