package workflow

import (
	"encoding/hex"
	"github.com/open-source-cloud/fuse/pkg/uuid"
	"strings"
)

// ExecID represents the ID for an execution of a workflow's node Function
type ExecID string

// NewExecID generates an UUIDv8 with embedded thread id for ExecID type
func NewExecID(thread uint16) ExecID {
	execUUID, _ := uuid.V8ExecID(thread)
	return ExecID(execUUID)
}

func (e ExecID) String() string {
	return string(e)
}

// Thread extracts the thread id from an ExecId
func (e ExecID) Thread() uint16 {
	// Remove hyphens for easy hex decoding
	cleaned := strings.ReplaceAll(string(e), "-", "")
	// Decode all 16 bytes
	raw, _ := hex.DecodeString(cleaned)

	// Extract threads parts:
	// uuid[6] (7th byte), uuid[7] (8th byte)
	hi := raw[6] & 0x0F          // low nibble of byte 6
	lo := raw[7]                 // all of byte 7
	threadIndex := (uint16(hi) << 8) | uint16(lo)
	return threadIndex
}
