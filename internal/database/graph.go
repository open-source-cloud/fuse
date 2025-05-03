package database

import (
	"context"
	"github.com/arangodb/go-driver/v2/arangodb"
	"github.com/open-source-cloud/fuse/internal/graph"
)

const (
	GraphSchemaCollectionName = "graph_schema"
)

// GraphSchemaCollection provides methods to manage collections of graphs within the connected ArangoDB instance.
type GraphSchemaCollection struct {
	database string
	client   *ArangoClient
}

// Create initializes a new graph within the collection using the specified name.
func (gc *GraphSchemaCollection) Create(ctx context.Context) error {
	db, err := gc.client.Database(ctx, gc.database)
	if err != nil {
		return err
	}

	_, err = db.CreateCollection(ctx, GraphSchemaCollectionName, &arangodb.CreateCollectionProperties{
		Type: arangodb.CollectionTypeDocument,
		Schema: &arangodb.CollectionSchemaOptions{
			Rule:  graph.JSONSchema(),
			Level: arangodb.CollectionSchemaLevelStrict,
		},
	})
	if err != nil {
		return err
	}

	return nil
}
