package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnvironmentValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		env     *Environment
		wantErr bool
	}{
		{name: "valid lowercase", env: NewEnvironment("staging", "Staging"), wantErr: false},
		{name: "valid with separators", env: NewEnvironment("prod-eu_1.2", ""), wantErr: false},
		{name: "default", env: NewEnvironment(DefaultEnvironmentName, ""), wantErr: false},
		{name: "empty name", env: NewEnvironment("", "x"), wantErr: true},
		{name: "uppercase rejected", env: NewEnvironment("Staging", ""), wantErr: true},
		{name: "leading separator rejected", env: NewEnvironment("-prod", ""), wantErr: true},
		{name: "spaces rejected", env: NewEnvironment("my env", ""), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.env.Validate()
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})
	}
}
