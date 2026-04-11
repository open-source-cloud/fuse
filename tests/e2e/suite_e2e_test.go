//go:build e2e

package e2e

import (
	"flag"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

// workflowsDirFlag is set from TestMain (-workflows); empty means use E2E_WORKFLOWS_DIR or repo default.
var workflowsDirFlag string

func TestMain(m *testing.M) {
	flag.StringVar(&workflowsDirFlag, "workflows", "", "directory of *.json workflow schemas (default: repo examples/workflows)")
	flag.Parse()
	os.Exit(m.Run())
}

var (
	e2eHealthOnce sync.Once
	e2eHealthErr  error
)

// RequireE2E returns an HTTP client and trimmed base URL, waiting for /health once per process.
func RequireE2E(t *testing.T) (*http.Client, string) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping e2e in short mode")
	}
	base := e2eBaseURL()
	client := NewHTTPClient()
	e2eHealthOnce.Do(func() {
		e2eHealthErr = WaitForHealth(client, base)
	})
	require.NoError(t, e2eHealthErr, "API should become healthy at %s", base)
	return client, base
}

func e2eBaseURL() string {
	u := os.Getenv("E2E_API_URL")
	if u == "" {
		u = DefaultAPIURL
	}
	return strings.TrimRight(u, "/")
}

// WorkflowsDirForTests resolves the examples workflows directory for schema files.
func WorkflowsDirForTests(t *testing.T) string {
	t.Helper()
	dir, err := ResolveWorkflowsDir(workflowsDirFlag)
	require.NoError(t, err, "resolve workflows directory")
	return dir
}

// E2EOverlayDir returns the e2e-specific overlay directory (examples/workflows/e2e/).
// Returns empty string if the directory does not exist.
func E2EOverlayDir(t *testing.T) string {
	t.Helper()
	base := WorkflowsDirForTests(t)
	overlay := filepath.Join(base, "e2e")
	if _, err := os.Stat(overlay); err == nil {
		return overlay
	}
	return ""
}
