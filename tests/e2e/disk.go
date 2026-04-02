package e2e

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ResolveWorkflowsDir returns an absolute workflows directory.
// If override is non-empty, it must exist as a directory.
// Otherwise E2E_WORKFLOWS_DIR is used if set.
// Otherwise examples/workflows under the repository root (go.mod walk from cwd) is used.
func ResolveWorkflowsDir(override string) (string, error) {
	if override != "" {
		abs, err := filepath.Abs(override)
		if err != nil {
			return "", err
		}
		if st, err := os.Stat(abs); err != nil || !st.IsDir() {
			return "", fmt.Errorf("workflows directory not found or not a directory: %s", abs)
		}
		return abs, nil
	}

	if d := os.Getenv("E2E_WORKFLOWS_DIR"); d != "" {
		abs, err := filepath.Abs(d)
		if err != nil {
			return "", err
		}
		if st, err := os.Stat(abs); err != nil || !st.IsDir() {
			return "", fmt.Errorf("E2E_WORKFLOWS_DIR is not a directory: %s", abs)
		}
		return abs, nil
	}

	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	root, err := findRepoRoot(wd)
	if err != nil {
		return "", err
	}
	dir := filepath.Join(root, "examples", "workflows")
	if st, err := os.Stat(dir); err != nil || !st.IsDir() {
		return "", fmt.Errorf("examples/workflows not found under repo root %s", root)
	}
	return dir, nil
}

func findRepoRoot(start string) (string, error) {
	dir := start
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errors.New("go.mod not found (run from repository root, set E2E_WORKFLOWS_DIR, or pass -workflows)")
		}
		dir = parent
	}
}

// ReadSchemaFile reads file only if it resolves under workflowsDir.
func ReadSchemaFile(workflowsDir, file string) ([]byte, error) {
	wd, err := filepath.Abs(workflowsDir)
	if err != nil {
		return nil, err
	}
	absFile, err := filepath.Abs(file)
	if err != nil {
		return nil, err
	}
	rel, err := filepath.Rel(wd, absFile)
	if err != nil || strings.HasPrefix(rel, "..") {
		return nil, fmt.Errorf("schema path outside workflows directory: %s", file)
	}
	// G304: absFile is constrained to workflowsDir by Rel check above.
	return os.ReadFile(absFile) //nolint:gosec // path validated against workflowsDir
}
