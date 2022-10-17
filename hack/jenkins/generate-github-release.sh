#!/usr/bin/env bash
set -e

cd "$(dirname "$-1")"

operator_yaml="deploy/kubernetes/wavefront-operator.yaml"

VERSION=$(cat ./release/OPERATOR_VERSION)
GITHUB_REPO=wavefrontHQ/wavefront-operator-for-kubernetes
AUTH="Authorization: token ${GITHUB_TOKEN}"

curl --fail -X POST -H "Content-Type:application/json" \
-H "$AUTH" \
-d "{
      \"tag_name\": \"v$VERSION\",
      \"target_commitish\": \"$GIT_BRANCH\",
      \"name\": \"Release v$VERSION\",
      \"body\": \"Description for v$VERSION\",
      \"draft\": true,
      \"prerelease\": false}" \
"https://api.github.com/repos/$GITHUB_REPO/releases"

id=$(curl -sH "$AUTH" "https://api.github.com/repos/$GITHUB_REPO/releases/tags/v${VERSION}" | jq ".id")

curl --data-binary @"$operator_yaml" \
  -H "$AUTH" \
  -H "Content-Type: application/octet-stream" \
"https://uploads.github.com/repos/$GITHUB_REPO/releases/$id/assets?name=$(basename $operator_yaml)"
