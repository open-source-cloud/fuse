package repositories

import (
	"errors"

	"github.com/open-source-cloud/fuse/pkg/workflow"
)

// ErrCredentialNotFound is returned when a credential is not found.
var ErrCredentialNotFound = errors.New("credential not found")

type (
	// CredentialRepository stores credential metadata (ADR-0031 Option B). Field VALUES are not
	// stored here; they live in the SecretStore at cred/<id>/<field>, per environment.
	CredentialRepository interface {
		FindByID(id string) (*workflow.Credential, error)
		FindAll() ([]*workflow.Credential, error)
		Save(cred *workflow.Credential) error
		Delete(id string) error
	}
)
