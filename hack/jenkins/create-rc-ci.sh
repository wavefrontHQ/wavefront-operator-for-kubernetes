#!/bin/bash -e
REPO_ROOT=$(git rev-parse --show-toplevel)

cd ${REPO_ROOT}

RELEASE_VERSION=$(cat ./release/OPERATOR_VERSION)
NEW_VERSION=$(semver-cli inc patch "$RELEASE_VERSION")
VERSION=$NEW_VERSION$VERSION_POSTFIX make generate-kubernetes-yaml

git fetch
git checkout .

git checkout $GIT_BRANCH

ls | grep -v build | xargs rm -rf
mv build/wavefront-operator.yaml wavefront-operator.yaml

git add --all .
git commit -m "updates wavefront-operator.yaml"
git push