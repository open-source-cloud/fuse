package uuid_test

import (
	"reflect"
	"testing"

	"github.com/open-source-cloud/fuse/pkg/uuid"
)

func TestV8ExecID(t *testing.T) {
	uuid, err := uuid.V8ExecID(1)
	if err != nil {
		t.Fatalf("failed to generate uuid: %s", err)
	}

	if len(uuid) != 36 {
		t.Fatalf("uuid should be 36 characters long, got %d", len(uuid))
	}

	if reflect.TypeOf(uuid) != reflect.TypeOf("") {
		t.Fatalf("uuid should be a string, got %s", reflect.TypeOf(uuid))
	}
}
