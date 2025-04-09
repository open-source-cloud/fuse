package uuid

import "github.com/google/uuid"

func V7() string {
	id, _ := uuid.NewV7()
	return id.String()
}
