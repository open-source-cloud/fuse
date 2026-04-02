#!/usr/bin/env bash
#
# k3s-teardown.sh — Delete the k3d development cluster.
#
set -euo pipefail

CLUSTER_NAME="${CLUSTER_NAME:-fuse-dev}"

echo "==> Checking prerequisites..."
if ! command -v k3d &>/dev/null; then
  echo "ERROR: 'k3d' is required but not found in PATH."
  exit 1
fi

if k3d cluster list 2>/dev/null | grep -q "${CLUSTER_NAME}"; then
  echo "==> Deleting k3d cluster '${CLUSTER_NAME}'..."
  k3d cluster delete "${CLUSTER_NAME}"
  echo "==> Cluster '${CLUSTER_NAME}' deleted."
else
  echo "==> Cluster '${CLUSTER_NAME}' does not exist, nothing to do."
fi
