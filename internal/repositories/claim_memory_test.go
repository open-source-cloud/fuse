package repositories_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-source-cloud/fuse/internal/repositories"
)

func TestMemoryClaimRepository_ClaimWorkflows_ReturnsEmpty(t *testing.T) {
	// Arrange
	repo := repositories.NewMemoryClaimRepository()

	// Act
	claimed, err := repo.ClaimWorkflows("node-1", 10)

	// Assert
	require.NoError(t, err)
	assert.Empty(t, claimed)
}

func TestMemoryClaimRepository_ClaimWorkflows_ConcurrentCallsNoPanic(_ *testing.T) {
	// Arrange
	repo := repositories.NewMemoryClaimRepository()
	done := make(chan struct{})

	// Act — simulate two nodes claiming simultaneously (no-op, but must not panic)
	go func() {
		_, _ = repo.ClaimWorkflows("node-1", 5)
		close(done)
	}()
	_, _ = repo.ClaimWorkflows("node-2", 5)
	<-done

	// Assert — if we reach here without panic/race, test passes
}

func TestMemoryClaimRepository_ReleaseWorkflows_NoError(t *testing.T) {
	// Arrange
	repo := repositories.NewMemoryClaimRepository()

	// Act
	err := repo.ReleaseWorkflows("node-1")

	// Assert
	require.NoError(t, err)
}

func TestMemoryClaimRepository_Heartbeat_NoError(t *testing.T) {
	// Arrange
	repo := repositories.NewMemoryClaimRepository()

	// Act
	err := repo.Heartbeat("node-1", "localhost", 9090)

	// Assert
	require.NoError(t, err)
}

func TestMemoryClaimRepository_FindStaleNodes_ReturnsEmpty(t *testing.T) {
	// Arrange
	repo := repositories.NewMemoryClaimRepository()

	// Act
	stale, err := repo.FindStaleNodes(30 * time.Second)

	// Assert
	require.NoError(t, err)
	assert.Empty(t, stale)
}

func TestMemoryClaimRepository_ReassignFromStaleNodes_ReturnsZero(t *testing.T) {
	tests := []struct {
		name     string
		nodeIDs  []string
		wantZero bool
	}{
		{
			name:     "empty list",
			nodeIDs:  []string{},
			wantZero: true,
		},
		{
			name:     "one stale node",
			nodeIDs:  []string{"dead-node-1"},
			wantZero: true,
		},
		{
			name:     "multiple stale nodes",
			nodeIDs:  []string{"dead-1", "dead-2", "dead-3"},
			wantZero: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			repo := repositories.NewMemoryClaimRepository()

			// Act
			released, err := repo.ReassignFromStaleNodes(tt.nodeIDs)

			// Assert
			require.NoError(t, err)
			if tt.wantZero {
				assert.Zero(t, released)
			}
		})
	}
}
