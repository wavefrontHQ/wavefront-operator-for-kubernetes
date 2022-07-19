#!/usr/bin/env bash
set -ex

cd "$(dirname "$0")" # cd to directory that create-bump-version-branch.sh is in

function print_usage_and_exit() {
  echo "Failure: $1"
  echo "Usage: $0 [flags] [options]"
  echo -e "\t-s semver component to bump for operator version (required)"
  echo -e "\t-c collector version. If not given, will be read from collector repo's VERSION file"
  exit 1
}

while getopts "s:c:" opt; do
  case $opt in
    s)
      OPERATOR_BUMP_COMPONENT="$OPTARG"
      ;;
    c)
      NEW_COLLECTOR_VERSION="$OPTARG"
      ;;
    \?)
      print_usage_and_exit "Invalid option: -$OPTARG"
      ;;
  esac
done

if [[ -z ${NEW_COLLECTOR_VERSION} ]] ; then
    NEW_COLLECTOR_VERSION=$(curl -s https://raw.githubusercontent.com/wavefrontHQ/wavefront-collector-for-kubernetes/main/release/VERSION)
fi

pushd ../../
  make semver-cli
popd

OLD_OPERATOR_VERSION=$(cat ../../release/OPERATOR_VERSION)
NEW_OPERATOR_VERSION=$(semver-cli inc "$OPERATOR_BUMP_COMPONENT" "$OLD_OPERATOR_VERSION")
echo "$NEW_OPERATOR_VERSION" >../../release/OPERATOR_VERSION

GIT_BUMP_BRANCH_NAME="bump-${NEW_OPERATOR_VERSION}"
git branch -D "$GIT_BUMP_BRANCH_NAME" &>/dev/null || true
git checkout -b "$GIT_BUMP_BRANCH_NAME"

OLD_COLLECTOR_VERSION=$(cat ../../release/COLLECTOR_VERSION)
echo "$NEW_COLLECTOR_VERSION" >../../release/COLLECTOR_VERSION

echo "Replacing collector version starting with ${OLD_COLLECTOR_VERSION} to ${NEW_COLLECTOR_VERSION} in ../../deploy/internal/2-wavefront-collector-daemonset.yaml"
sed -i '' "s/${OLD_COLLECTOR_VERSION}.*/${NEW_COLLECTOR_VERSION}/g" "../../deploy/internal/2-wavefront-collector-daemonset.yaml"
echo "Replacing collector version starting with ${OLD_COLLECTOR_VERSION} to ${NEW_COLLECTOR_VERSION} in ../../deploy/internal/2-wavefront-collector-deployment.yaml"
sed -i '' "s/${OLD_COLLECTOR_VERSION}.*/${NEW_COLLECTOR_VERSION}/g" "../../deploy/internal/2-wavefront-collector-deployment.yaml"

git commit -am "Bump operator version to ${NEW_OPERATOR_VERSION} and collector version to ${NEW_COLLECTOR_VERSION}"

#git push --force --set-upstream origin "${GIT_BUMP_BRANCH_NAME}"