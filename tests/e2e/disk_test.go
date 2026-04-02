package e2e

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveWorkflowsDir_override(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	got, err := ResolveWorkflowsDir(dir)
	require.NoError(t, err)
	assert.Equal(t, dir, got)
}

func TestResolveWorkflowsDir_invalidOverride(t *testing.T) {
	t.Parallel()
	_, err := ResolveWorkflowsDir("/nonexistent/workflows/dir/e2e-test-xyz")
	require.Error(t, err)
}

func TestResolveWorkflowsDir_repoRelative(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "go.mod"), []byte("module e2e_test_root\n"), 0o600))
	wf := filepath.Join(root, "examples", "workflows")
	require.NoError(t, os.MkdirAll(wf, 0o750))

	prev, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(root))
	t.Cleanup(func() { _ = os.Chdir(prev) })

	t.Setenv("E2E_WORKFLOWS_DIR", "")

	got, err := ResolveWorkflowsDir("")
	require.NoError(t, err)
	wantResolved, err := filepath.EvalSymlinks(wf)
	require.NoError(t, err)
	gotResolved, err := filepath.EvalSymlinks(got)
	require.NoError(t, err)
	assert.Equal(t, wantResolved, gotResolved)
}

func TestReadSchemaFile_rejectsOutsideDir(t *testing.T) {
	t.Parallel()
	wf := t.TempDir()
	outside := t.TempDir()
	bad := filepath.Join(outside, "x.json")
	require.NoError(t, os.WriteFile(bad, []byte("{}"), 0o600))

	_, err := ReadSchemaFile(wf, bad)
	require.Error(t, err)
}
