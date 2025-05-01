package enginemsg

import "github.com/open-source-cloud/fuse/internal/actormodel"

// StartWorkflow start a new workflow worker in the engine
const StartWorkflow actormodel.MessageType = "engine:start:workflow"
