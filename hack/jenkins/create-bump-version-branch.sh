#!/usr/bin/env bash
set -ex

cd "$(dirname "$0")" # cd to directory that create-bump-version-branch.sh is in

function print_usage_and_exit() {
  echo "Failure: $1"
  echo "Usage: $0 [flags] [options]"
  echo -e "\t-s semver component to bump for operator version (required)"
  exit 1
}

while getopts "s:" opt; do
  case $opt in
    s)
      OPERATOR_BUMP_COMPONENT="$OPTARG"
      ;;
    \?)
      print_usage_and_exit "Invalid option: -$OPTARG"
      ;;
  esac
done

pushd ../../
  make semver-cli
popd

OLD_OPERATOR_VERSION=$(cat ../../release/OPERATOR_VERSION)
NEW_OPERATOR_VERSION=$(semver-cli inc "$OPERATOR_BUMP_COMPONENT" "$OLD_OPERATOR_VERSION")
echo "$NEW_OPERATOR_VERSION" >../../release/OPERATOR_VERSION

GIT_BUMP_BRANCH_NAME="bump-${NEW_OPERATOR_VERSION}"
git branch -D "$GIT_BUMP_BRANCH_NAME" &>/dev/null || true
git checkout -b "$GIT_BUMP_BRANCH_NAME"

git commit -am "Bump operator version to ${NEW_OPERATOR_VERSION}"

#git push --force --set-upstream origin "${GIT_BUMP_BRANCH_NAME}"