package workflow

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/open-source-cloud/fuse/pkg/secrets"
	"github.com/open-source-cloud/fuse/pkg/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkflow_ResolveSecretReferences(t *testing.T) {
	store := secrets.NewMemorySecretStore()
	require.NoError(t, store.Set(context.Background(), secrets.Scope{Environment: "test"}, "tok", "T0KEN"))

	w := &Workflow{id: workflow.ID("wf-1")}
	w.SetSecretResolver(secrets.NewResolver(store, "test"))

	// {{secret:NAME}} embedded in a schema string -> a SecretValue wrapping the whole value.
	v, err := w.resolveSchemaValue("Bearer {{secret:tok}}")
	require.NoError(t, err)
	sv, ok := v.(secrets.SecretValue)
	require.True(t, ok)
	assert.Equal(t, "Bearer T0KEN", sv.Reveal())
	b, _ := json.Marshal(sv)
	assert.JSONEq(t, `"***"`, string(b))

	// A reference-free value passes through unchanged.
	v, err = w.resolveSchemaValue("plain")
	require.NoError(t, err)
	assert.Equal(t, "plain", v)

	// source:"secret" resolves to a SecretValue.
	sv2, err := w.resolveSecret("tok")
	require.NoError(t, err)
	assert.Equal(t, "T0KEN", sv2.Reveal())

	// An unknown secret surfaces the not-found error.
	_, err = w.resolveSecret("missing")
	assert.ErrorIs(t, err, secrets.ErrSecretNotFound)

	// With no resolver configured, a secret reference is an error.
	wNoResolver := &Workflow{id: workflow.ID("wf-2")}
	_, err = wNoResolver.resolveSecret("tok")
	assert.Error(t, err)
}
