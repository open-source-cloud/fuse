#!/usr/bin/env bash
#
# kind-setup.sh — Create a kind cluster, build the FUSE image, load it, and install via Helm.
# Requires: kind, docker, kubectl, helm (Docker Desktop + kind work on macOS/Windows/Linux).
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
KIND_CONFIG="${PROJECT_ROOT}/deploy/kind/cluster-config.yaml"

echo "==> Checking prerequisites..."
for cmd in kind docker kubectl helm; do
  if ! command -v "${cmd}" &>/dev/null; then
    echo "ERROR: '${cmd}' is required but not found in PATH." >&2
    exit 1
  fi
done

echo "==> Creating kind cluster '${CLUSTER_NAME}' (if missing)..."
if kind get clusters 2>/dev/null | grep -qx "${CLUSTER_NAME}"; then
  echo "    Cluster '${CLUSTER_NAME}' already exists, skipping creation."
else
  kind create cluster --name "${CLUSTER_NAME}" --config "${KIND_CONFIG}" --wait 120s
fi

echo "==> Building Docker image '${IMAGE_NAME}'..."
docker build -t "${IMAGE_NAME}" "${PROJECT_ROOT}"

echo "==> Loading image into kind cluster..."
kind load docker-image "${IMAGE_NAME}" --name "${CLUSTER_NAME}"

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
echo "    OpenAPI/Swagger UI is served at /docs (spec is generated during docker build)."
echo "    To reach the API from your machine:"
echo "      kubectl -n ${NAMESPACE} port-forward svc/${RELEASE_NAME} 9090:9090"
echo "      curl -sS http://127.0.0.1:9090/health"
echo "      open http://127.0.0.1:9090/docs   # Swagger UI"
