#!/usr/bin/env bash
# Bump the release version across all deploy manifests.
#
# Usage: ./scripts/bump-version.sh <VERSION>
# Example: ./scripts/bump-version.sh 0.3.0
#
# Files updated:
#   deploy/helm/fuse/Chart.yaml  — version + appVersion
#   deploy/helm/fuse/values.yaml — image.tag

set -euo pipefail

if [ $# -ne 1 ]; then
    echo "Usage: $0 <VERSION>" >&2
    echo "Example: $0 0.3.0" >&2
    exit 1
fi

VERSION="$1"

if ! echo "$VERSION" | grep -qE '^[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9.]+)?$'; then
    echo "Error: VERSION must be semver (e.g. 1.2.3 or 1.2.3-rc.1)" >&2
    exit 1
fi

REPO_ROOT=$(git rev-parse --show-toplevel 2>/dev/null || pwd)

CHART="$REPO_ROOT/deploy/helm/fuse/Chart.yaml"
VALUES="$REPO_ROOT/deploy/helm/fuse/values.yaml"

for f in "$CHART" "$VALUES"; do
    if [ ! -f "$f" ]; then
        echo "Error: file not found: $f" >&2
        exit 1
    fi
done

# Chart.yaml — version (chart version)
sed -i '' "s/^version: .*/version: ${VERSION}/" "$CHART"

# Chart.yaml — appVersion (container image tag)
sed -i '' "s/^appVersion: .*/appVersion: \"${VERSION}\"/" "$CHART"

# values.yaml — image.tag
sed -i '' "s/^  tag: .*/  tag: \"${VERSION}\"/" "$VALUES"

echo "Bumped to ${VERSION}:"
echo "  $CHART  — version + appVersion"
echo "  $VALUES — image.tag"
