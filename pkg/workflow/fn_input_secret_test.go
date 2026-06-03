package workflow_test

import (
	"encoding/json"
	"testing"

	"github.com/open-source-cloud/fuse/pkg/secrets"
	"github.com/open-source-cloud/fuse/pkg/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFunctionInput_SecretRedaction(t *testing.T) {
	t.Parallel()
	in, err := workflow.NewFunctionInputWith(map[string]any{
		"apiKey": secrets.NewSecretValue("s3cr3t"),
		"plain":  "visible",
	})
	require.NoError(t, err)

	// The consuming function obtains the plaintext via GetStr.
	assert.Equal(t, "s3cr3t", in.GetStr("apiKey"))
	assert.Equal(t, "visible", in.GetStr("plain"))

	// But anywhere the input map is serialized (journal / snapshot / trace / logs)
	// the secret is redacted.
	b, err := json.Marshal(in.Raw())
	require.NoError(t, err)
	assert.Contains(t, string(b), `"***"`)
	assert.NotContains(t, string(b), "s3cr3t")
	assert.Contains(t, string(b), "visible")
}
