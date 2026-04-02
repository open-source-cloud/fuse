#!/usr/bin/env bash
#
# k3s-setup.sh — Create a k3d cluster, build the FUSE image, import it, and install via Helm.
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

echo "==> Checking prerequisites..."
for cmd in k3d docker helm; do
  if ! command -v "${cmd}" &>/dev/null; then
    echo "ERROR: '${cmd}' is required but not found in PATH."
    exit 1
  fi
done

echo "==> Creating k3d cluster '${CLUSTER_NAME}'..."
if k3d cluster list | grep -q "${CLUSTER_NAME}"; then
  echo "    Cluster '${CLUSTER_NAME}' already exists, skipping creation."
else
  k3d cluster create "${CLUSTER_NAME}" \
    --port "9090:80@loadbalancer" \
    --agents 1 \
    --wait
fi

echo "==> Building Docker image '${IMAGE_NAME}'..."
docker build -t "${IMAGE_NAME}" "${PROJECT_ROOT}"

echo "==> Importing image into k3d cluster..."
k3d image import "${IMAGE_NAME}" --cluster "${CLUSTER_NAME}"

echo "==> Updating Helm dependencies..."
helm dependency update "${CHART_DIR}"

echo "==> Creating namespace '${NAMESPACE}' (if needed)..."
kubectl create namespace "${NAMESPACE}" --dry-run=client -o yaml | kubectl apply -f -

echo "==> Installing/upgrading Helm release '${RELEASE_NAME}'..."
helm upgrade --install "${RELEASE_NAME}" "${CHART_DIR}" \
  --namespace "${NAMESPACE}" \
  --values "${VALUES_FILE}" \
  --wait \
  --timeout 5m

echo ""
echo "==> Done! FUSE is deployed in the '${NAMESPACE}' namespace."
echo "    To access the API:"
echo "      kubectl -n ${NAMESPACE} port-forward svc/${RELEASE_NAME}-fuse 9090:9090"
echo "      curl http://127.0.0.1:9090/health"
