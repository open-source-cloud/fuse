package repositories

import "time"

// MemoryClaimRepository is a no-op stub used when HA is disabled.
type MemoryClaimRepository struct{}

// NewMemoryClaimRepository creates a new no-op claim repository.
func NewMemoryClaimRepository() *MemoryClaimRepository {
	return &MemoryClaimRepository{}
}

// ClaimWorkflows is a no-op in memory mode.
func (r *MemoryClaimRepository) ClaimWorkflows(_ string, _ int) ([]ClaimedWorkflow, error) {
	return nil, nil
}

// ReleaseWorkflows is a no-op in memory mode.
func (r *MemoryClaimRepository) ReleaseWorkflows(_ string) error {
	return nil
}

// Heartbeat is a no-op in memory mode.
func (r *MemoryClaimRepository) Heartbeat(_ string, _ string, _ int) error {
	return nil
}

// FindStaleNodes is a no-op in memory mode.
func (r *MemoryClaimRepository) FindStaleNodes(_ time.Duration) ([]string, error) {
	return nil, nil
}

// ReassignFromStaleNodes is a no-op in memory mode.
func (r *MemoryClaimRepository) ReassignFromStaleNodes(_ []string) (int, error) {
	return 0, nil
}
