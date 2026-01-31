#!/usr/bin/env bash
# Pre-commit quality gates script
# Enforces: Lint → Build → Test

set -e

echo "🔍 Running quality gates..."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Get repository root
REPO_ROOT=$(git rev-parse --show-toplevel 2>/dev/null || pwd)
cd "$REPO_ROOT"

# Gate 1: Lint
echo ""
echo "📋 Gate 1: Linting..."
if make lint > /dev/null 2>&1; then
    echo -e "${GREEN}✓ Linting passed${NC}"
else
    echo -e "${RED}✗ Linting failed${NC}"
    echo ""
    echo "Run 'make lint' to see errors, or 'make lint-fix' to auto-fix issues"
    exit 1
fi

# Gate 2: Build
echo ""
echo "🔨 Gate 2: Building..."
if make build > /dev/null 2>&1; then
    echo -e "${GREEN}✓ Build succeeded${NC}"
else
    echo -e "${RED}✗ Build failed${NC}"
    echo ""
    echo "Run 'make build' to see errors"
    exit 1
fi

# Gate 3: Test
echo ""
echo "🧪 Gate 3: Testing..."
if make test > /dev/null 2>&1; then
    echo -e "${GREEN}✓ Tests passed${NC}"
else
    echo -e "${RED}✗ Tests failed${NC}"
    echo ""
    echo "Run 'make test' to see test failures"
    exit 1
fi

echo ""
echo -e "${GREEN}✅ All quality gates passed!${NC}"
echo ""
