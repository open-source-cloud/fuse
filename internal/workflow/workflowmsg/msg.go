// Package workflowmsg workflow actor messages
package workflowmsg

import "github.com/open-source-cloud/fuse/internal/actormodel"

const (
	// Start a message that starts a workflow execution
	Start actormodel.MessageType = "workflow:start"
	// Continue a message that continues a workflow execution
	Continue actormodel.MessageType = "workflow:continue"
)
