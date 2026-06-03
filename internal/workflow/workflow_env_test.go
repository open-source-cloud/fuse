package workflow

import (
	"testing"

	pkgwf "github.com/open-source-cloud/fuse/pkg/workflow"
	"github.com/stretchr/testify/assert"
)

func TestNewStoresEnvironment(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		environment string
		want        string
	}{
		{name: "explicit environment", environment: "staging", want: "staging"},
		{name: "default environment", environment: "default", want: "default"},
		{name: "empty environment", environment: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			w := New(pkgwf.ID("wf-env"), nil, tt.environment)

			assert.Equal(t, tt.want, w.Environment())
		})
	}
}
