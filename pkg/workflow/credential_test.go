package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCredentialValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cred    *Credential
		wantErr bool
	}{
		{name: "valid", cred: NewCredential("openai-prod", "openai", "Prod key", []string{"apiKey"}), wantErr: false},
		{name: "dotted id ok", cred: NewCredential("my.org.openai", "openai", "", []string{"apiKey", "baseUrl"}), wantErr: false},
		{name: "no fields ok", cred: NewCredential("empty", "custom", "", nil), wantErr: false},
		{name: "missing id", cred: NewCredential("", "openai", "", []string{"apiKey"}), wantErr: true},
		{name: "missing type", cred: NewCredential("c1", "", "", []string{"apiKey"}), wantErr: true},
		{name: "uppercase id rejected", cred: NewCredential("Prod", "openai", "", nil), wantErr: true},
		{name: "bad field name rejected", cred: NewCredential("c1", "openai", "", []string{"api Key"}), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.cred.Validate()
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})
	}
}
