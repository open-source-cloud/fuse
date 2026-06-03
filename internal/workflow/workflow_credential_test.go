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

func TestWorkflow_ResolveCredentialReferences(t *testing.T) {
	store := secrets.NewMemorySecretStore()
	// Credential field values live at the reserved cred/<id>/<field> secret name.
	require.NoError(t, store.Set(context.Background(), secrets.Scope{Environment: "test"},
		secrets.CredentialSecretName("openai-prod", "apiKey"), "sk-PROD"))

	w := &Workflow{id: workflow.ID("wf-1")}
	w.SetSecretResolver(secrets.NewResolver(store, "test"))

	// source:"credential" with "<id>.<field>" resolves to a redacted SecretValue.
	sv, err := w.resolveCredential("openai-prod.apiKey")
	require.NoError(t, err)
	assert.Equal(t, "sk-PROD", sv.Reveal())
	b, _ := json.Marshal(sv)
	assert.JSONEq(t, `"***"`, string(b))

	// {{credential:id.field}} embedded in a schema string -> a SecretValue wrapping the whole value.
	v, err := w.resolveSchemaValue("Bearer {{credential:openai-prod.apiKey}}")
	require.NoError(t, err)
	wrapped, ok := v.(secrets.SecretValue)
	require.True(t, ok)
	assert.Equal(t, "Bearer sk-PROD", wrapped.Reveal())

	// Malformed reference (no field) is an error.
	_, err = w.resolveCredential("openai-prod")
	require.Error(t, err)

	// Unknown credential field surfaces the not-found error.
	_, err = w.resolveCredential("openai-prod.missing")
	assert.ErrorIs(t, err, secrets.ErrSecretNotFound)
}
