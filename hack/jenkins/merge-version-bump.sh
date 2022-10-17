#!/usr/bin/env bash
set -e

cd "$(dirname "$0")"

VERSION=$(cat ../../release/OPERATOR_VERSION)
GIT_BUMP_BRANCH_NAME="bump-${VERSION}"
git branch -D "$GIT_BUMP_BRANCH_NAME" &>/dev/null || true
git checkout -b "$GIT_BUMP_BRANCH_NAME"

git commit -am "Bump operator version to ${VERSION}"
git push --force --set-upstream origin "${GIT_BUMP_BRANCH_NAME}"

PR_URL=$(curl \
  -X POST \
  -H "Authorization: token ${GITHUB_TOKEN}" \
  -d "{\"head\":\"${GIT_BUMP_BRANCH_NAME}\",\"base\":\"main\",\"title\":\"Bump version to ${VERSION}\"}" \
  https://api.github.com/repos/wavefrontHQ/wavefront-operator-for-kubernetes/pulls |
  jq -r '.url')

echo "PR URL: ${PR_URL}"

curl \
  -X PUT \
  -H "Authorization: token ${GITHUB_TOKEN}" \
  -H "Accept: application/vnd.github.v3+json" \
  "${PR_URL}/merge" \
  -d "{\"commit_title\":\"Bump version to ${VERSION}\", \"merge_method\":\"squash\"}"