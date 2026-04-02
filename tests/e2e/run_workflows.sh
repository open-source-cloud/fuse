#!/usr/bin/env bash
set -euo pipefail

API_URL="${E2E_API_URL:-http://localhost:9090}"
WORKFLOWS_DIR="$(cd "$(dirname "$0")/../../examples/workflows" && pwd)"

passed=0
failed=0
errors=()

# Wait for API to be ready
echo "Waiting for API at ${API_URL}/health ..."
retries=30
until curl -sf "${API_URL}/health" > /dev/null 2>&1; do
  retries=$((retries - 1))
  if [[ "$retries" -le 0 ]]; then
    echo "ERROR: API did not become ready within timeout" >&2
    exit 1
  fi
  sleep 2
done
echo "API is ready."
echo ""

for file in "${WORKFLOWS_DIR}"/*.json; do
  schema_id="$(basename "${file}" .json)"
  echo "--- Testing schema: ${schema_id} ---"

  # PUT the schema
  put_status=$(curl -s -o /dev/null -w "%{http_code}" \
    -X PUT \
    -H "Content-Type: application/json" \
    -d @"${file}" \
    "${API_URL}/v1/schemas/${schema_id}")

  if [[ "${put_status}" -lt 200 || "${put_status}" -ge 300 ]]; then
    echo "  FAIL: PUT /v1/schemas/${schema_id} returned HTTP ${put_status}"
    failed=$((failed + 1))
    errors+=("${schema_id}: PUT schema returned HTTP ${put_status}")
    echo ""
    continue
  fi
  echo "  PUT schema: HTTP ${put_status} OK"

  # POST trigger
  trigger_response=$(curl -s -w "\n%{http_code}" \
    -X POST \
    -H "Content-Type: application/json" \
    -d "{\"schemaID\": \"${schema_id}\"}" \
    "${API_URL}/v1/workflows/trigger")

  trigger_body=$(echo "${trigger_response}" | head -n -1)
  trigger_status=$(echo "${trigger_response}" | tail -n 1)

  if [[ "${trigger_status}" -lt 200 || "${trigger_status}" -ge 300 ]]; then
    echo "  FAIL: POST /v1/workflows/trigger returned HTTP ${trigger_status}"
    failed=$((failed + 1))
    errors+=("${schema_id}: trigger returned HTTP ${trigger_status}")
    echo ""
    continue
  fi

  # Check for workflowId in response
  workflow_id=$(echo "${trigger_body}" | grep -o '"workflowId"\s*:\s*"[^"]*"' | head -1 | cut -d'"' -f4)

  if [[ -z "${workflow_id}" ]]; then
    echo "  FAIL: trigger response missing workflowId"
    echo "  Response: ${trigger_body}"
    failed=$((failed + 1))
    errors+=("${schema_id}: trigger response missing workflowId")
    echo ""
    continue
  fi

  echo "  Trigger: HTTP ${trigger_status} OK (workflowId=${workflow_id})"
  passed=$((passed + 1))
  echo ""
done

# Summary
echo "==============================="
echo "E2E Test Summary"
echo "==============================="
echo "Passed: ${passed}"
echo "Failed: ${failed}"
echo "Total:  $((passed + failed))"

if [[ ${#errors[@]} -gt 0 ]]; then
  echo ""
  echo "Failures:"
  for err in "${errors[@]}"; do
    echo "  - ${err}"
  done
fi

echo "==============================="

if [[ "${failed}" -gt 0 ]]; then
  exit 1
fi
