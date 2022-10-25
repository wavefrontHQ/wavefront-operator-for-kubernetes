#!/bin/bash -e
cd "$(git rev-parse --show-toplevel)"

git config --global user.email "svc.wf-jenkins@vmware.com"
git config --global user.name "svc.wf-jenkins"
git remote set-url origin https://${TOKEN}@github.com/wavefronthq/wavefront-operator-for-kubernetes.git

RELEASE_VERSION=$(cat ./release/OPERATOR_VERSION)
NEW_VERSION=$(semver-cli inc patch "$RELEASE_VERSION")

VERSION=$NEW_VERSION$VERSION_POSTFIX make generate-kubernetes-yaml
cp deploy/kubernetes/wavefront-operator.yaml build/wavefront-operator.yaml

git checkout .
git fetch
git checkout rc
git reset --hard origin/rc

git clean -dfx -e build
OPERATOR_FILE="wavefront-operator-${GIT_BRANCH}.yaml"
mv build/wavefront-operator.yaml "$OPERATOR_FILE"

git add --all .
git commit -m "build $OPERATOR_FILE from $GIT_COMMIT" || exit 0
git push origin rc || exit 0
