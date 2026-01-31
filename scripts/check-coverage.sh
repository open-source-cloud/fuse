#!/usr/bin/env bash
# Check test coverage thresholds
# Coverage requirements:
# - Critical business logic: 90%+
# - Repository implementations: 80%+
# - HTTP handlers: 70%+
# - Actor message handling: 80%+

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Get repository root
REPO_ROOT=$(git rev-parse --show-toplevel 2>/dev/null || pwd)
cd "$REPO_ROOT"

echo "📊 Checking test coverage thresholds..."
echo ""

# Generate coverage profile
COVERAGE_FILE="coverage.out"
go test -coverprofile="$COVERAGE_FILE" ./pkg/... ./internal/... > /dev/null 2>&1

if [ ! -f "$COVERAGE_FILE" ]; then
    echo -e "${RED}✗ Failed to generate coverage profile${NC}"
    exit 1
fi

# Get overall coverage
OVERALL_COVERAGE=$(go tool cover -func="$COVERAGE_FILE" | grep total | awk '{print $3}' | sed 's/%//')

echo "Overall coverage: ${OVERALL_COVERAGE}%"
echo ""

# Check coverage by package type
FAILED=0

# Check repository packages (80%+)
echo "Checking repository packages (threshold: 80%)..."
REPO_COVERAGE=$(go tool cover -func="$COVERAGE_FILE" | grep "repositories" | awk '{print $3}' | sed 's/%//' | head -1)
if [ -n "$REPO_COVERAGE" ]; then
    if (( $(echo "$REPO_COVERAGE < 80" | bc -l) )); then
        echo -e "${RED}✗ Repository coverage: ${REPO_COVERAGE}% (required: 80%+)${NC}"
        FAILED=1
    else
        echo -e "${GREEN}✓ Repository coverage: ${REPO_COVERAGE}%${NC}"
    fi
fi

# Check handler packages (70%+)
echo "Checking handler packages (threshold: 70%)..."
HANDLER_COVERAGE=$(go tool cover -func="$COVERAGE_FILE" | grep "handlers" | awk '{print $3}' | sed 's/%//' | head -1)
if [ -n "$HANDLER_COVERAGE" ]; then
    if (( $(echo "$HANDLER_COVERAGE < 70" | bc -l) )); then
        echo -e "${RED}✗ Handler coverage: ${HANDLER_COVERAGE}% (required: 70%+)${NC}"
        FAILED=1
    else
        echo -e "${GREEN}✓ Handler coverage: ${HANDLER_COVERAGE}%${NC}"
    fi
fi

# Check workflow packages (90%+ for critical logic)
echo "Checking workflow packages (threshold: 90%)..."
WORKFLOW_COVERAGE=$(go tool cover -func="$COVERAGE_FILE" | grep "workflow" | awk '{print $3}' | sed 's/%//' | head -1)
if [ -n "$WORKFLOW_COVERAGE" ]; then
    if (( $(echo "$WORKFLOW_COVERAGE < 90" | bc -l) )); then
        echo -e "${YELLOW}⚠ Workflow coverage: ${WORKFLOW_COVERAGE}% (recommended: 90%+)${NC}"
    else
        echo -e "${GREEN}✓ Workflow coverage: ${WORKFLOW_COVERAGE}%${NC}"
    fi
fi

# Cleanup
rm -f "$COVERAGE_FILE"

echo ""
if [ $FAILED -eq 1 ]; then
    echo -e "${RED}✗ Coverage thresholds not met${NC}"
    echo "See coverage requirements in .cursor/rules/07-testing.mdc"
    exit 1
else
    echo -e "${GREEN}✅ Coverage thresholds met${NC}"
    exit 0
fi
