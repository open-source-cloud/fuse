#!/usr/bin/env bash
#
# kind-redeploy.sh — Rebuild the image, load it into kind, helm upgrade, and restart workloads.
# Use after handler/API changes so Swagger (generated in Dockerfile) and the binary stay in sync.
#
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

CLUSTER_NAME="${CLUSTER_NAME:-fuse-dev}"
RELEASE_NAME="${RELEASE_NAME:-fuse}"
NAMESPACE="${NAMESPACE:-fuse}"
IMAGE_NAME="fuse-app:dev"
VALUES_FILE="${PROJECT_ROOT}/deploy/k3s/values-dev.yaml"
CHART_DIR="${PROJECT_ROOT}/deploy/helm/fuse"

for cmd in kind docker kubectl helm; do
  if ! command -v "${cmd}" &>/dev/null; then
    echo "ERROR: '${cmd}' is required but not found in PATH." >&2
    exit 1
  fi
done

if ! kind get clusters 2>/dev/null | grep -qx "${CLUSTER_NAME}"; then
  echo "ERROR: kind cluster '${CLUSTER_NAME}' not found. Run: make kind-deploy" >&2
  exit 1
fi

echo "==> Rebuilding ${IMAGE_NAME} (runs swag init inside Docker for /docs)..."
# Set FUSE_DOCKER_BUILD_NO_CACHE=1 if the image layers cache a stale OpenAPI embed.
docker build ${FUSE_DOCKER_BUILD_NO_CACHE:+--no-cache} -t "${IMAGE_NAME}" "${PROJECT_ROOT}"

echo "==> Loading image into kind..."
kind load docker-image "${IMAGE_NAME}" --name "${CLUSTER_NAME}"

echo "==> Helm upgrade..."
helm dependency update "${CHART_DIR}"
helm upgrade --install "${RELEASE_NAME}" "${CHART_DIR}" \
  --namespace "${NAMESPACE}" \
  --values "${VALUES_FILE}" \
  --wait \
  --timeout 5m

echo "==> Restarting pods so they pick up the loaded image (same tag)..."
kubectl rollout restart "deployment/${RELEASE_NAME}" -n "${NAMESPACE}"
kubectl rollout status "deployment/${RELEASE_NAME}" -n "${NAMESPACE}" --timeout=3m

echo ""
echo "==> Redeploy complete. Port-forward and open Swagger:"
echo "    kubectl -n ${NAMESPACE} port-forward svc/${RELEASE_NAME} 9090:9090"
echo "    open http://127.0.0.1:9090/docs"
