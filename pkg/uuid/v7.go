// Package uuid provides helpers for working with UUIDs.
package uuid

import "github.com/google/uuid"

// V7 generates a new UUIDv7 and returns directly, ignoring errors
func V7() string {
	id, _ := uuid.NewV7()
	return id.String()
}
