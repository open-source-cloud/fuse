package services

import (
	"testing"

	"github.com/open-source-cloud/fuse/internal/repositories"
	"github.com/open-source-cloud/fuse/pkg/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newEnvironmentService() EnvironmentService {
	return NewEnvironmentService(repositories.NewMemoryEnvironmentRepository())
}

func TestEnvironmentService_IsValid(t *testing.T) {
	t.Parallel()

	svc := newEnvironmentService()
	_, err := svc.Save(workflow.NewEnvironment("staging", "Staging"))
	require.NoError(t, err)

	tests := []struct {
		name string
		env  string
		want bool
	}{
		{name: "default always valid", env: workflow.DefaultEnvironmentName, want: true},
		{name: "declared environment valid", env: "staging", want: true},
		{name: "unknown environment invalid", env: "bogus", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, svc.IsValid(tt.env))
		})
	}
}

func TestEnvironmentService_SaveRejectsInvalidName(t *testing.T) {
	t.Parallel()

	svc := newEnvironmentService()

	_, err := svc.Save(workflow.NewEnvironment("Invalid Name", ""))

	require.Error(t, err)
}

func TestEnvironmentService_CRUDRoundTrip(t *testing.T) {
	t.Parallel()

	svc := newEnvironmentService()

	saved, err := svc.Save(workflow.NewEnvironment("prod", "Production"))
	require.NoError(t, err)
	assert.Equal(t, "prod", saved.Name)

	found, err := svc.FindByID("prod")
	require.NoError(t, err)
	assert.Equal(t, "Production", found.Description)

	require.NoError(t, svc.Delete("prod"))
	assert.False(t, svc.IsValid("prod"))
}
