package repositories

import "time"

// ClaimedWorkflow represents a workflow claimed by an HA node.
type ClaimedWorkflow struct {
	WorkflowID string
	SchemaID   string
	State      string
}

// ClaimRepository manages HA workflow claiming and node heartbeats.
type ClaimRepository interface {
	// ClaimWorkflows atomically claims unclaimed or stale-lease workflows for the given node.
	// Returns up to limit claimed workflows.
	ClaimWorkflows(nodeID string, limit int) ([]ClaimedWorkflow, error)

	// ReleaseWorkflows releases all workflows claimed by the given node.
	ReleaseWorkflows(nodeID string) error

	// Heartbeat upserts the node's heartbeat record.
	Heartbeat(nodeID string, host string, port int) error

	// FindStaleNodes returns node IDs whose last heartbeat is older than the given timeout.
	FindStaleNodes(timeout time.Duration) ([]string, error)

	// ReassignFromStaleNodes releases workflows claimed by stale nodes so they can be reclaimed.
	// Returns the number of workflows released.
	ReassignFromStaleNodes(staleNodeIDs []string) (int, error)
}
