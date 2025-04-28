// Package database provides a database client
package database

import (
	"context"
	"fmt"
	"github.com/arangodb/go-driver/v2/arangodb"
	"github.com/arangodb/go-driver/v2/connection"
	"github.com/open-source-cloud/fuse/internal/audit"
	"strings"
)

// ArangoClient is a arango database client wrapper
type ArangoClient struct {
	arangodb.Client
}

// NewClient creates a new database client
func NewClient(host, port, user, pass string, tls bool) (*ArangoClient, error) {
	// TODO: Support TLS and async connections
	var skipVerify bool = true
	var protocol string = "http"
	if tls {
		protocol = "https"
		skipVerify = false
	}

	// Trim all inputs to avoid whitespace issues
	host = strings.TrimSpace(host)
	port = strings.TrimSpace(port)
	user = strings.TrimSpace(user)
	pass = strings.TrimSpace(pass)

	endpoint := fmt.Sprintf("%s://%s:%s", protocol, host, port)

	audit.Debug().Msgf("connecting to arango endpoint: %s", endpoint)
	audit.Debug().Msgf("using credentials with username: %s", user)

	// Create a connection configuration with authentication included
	conn := connection.NewHttp2Connection(
		connection.DefaultHTTP2ConfigurationWrapper(
			connection.NewRoundRobinEndpoints([]string{endpoint}), skipVerify),
	)

	// Set basic auth credentials
	auth := connection.NewBasicAuth(user, pass)
	err := conn.SetAuthentication(auth)
	if err != nil {
		return nil, fmt.Errorf("failed to set authentication: %w", err)
	}

	cl := arangodb.NewClient(conn)

	audit.Debug().Msgf("connected to arango endpoint: %s", endpoint)

	return &ArangoClient{cl}, nil
}

// Ping checks the connectivity to the database and returns an error if the connection is not established or fails.
func (c *ArangoClient) Ping() error {
	ctx := context.Background()
	_, err := c.Version(ctx)
	if err != nil {
		return fmt.Errorf("failed to connect to ArangoDB: %w", err)
	}
	return nil
}
