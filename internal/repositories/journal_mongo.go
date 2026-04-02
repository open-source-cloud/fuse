package repositories

import (
	"context"
	"errors"
	"time"

	"github.com/open-source-cloud/fuse/internal/app/config"
	internalworkflow "github.com/open-source-cloud/fuse/internal/workflow"
	"github.com/open-source-cloud/fuse/pkg/strutil"
	"github.com/open-source-cloud/fuse/pkg/workflow"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var (
	// ErrJournalMongoNotImplemented is returned when a journal operation is not implemented
	ErrJournalMongoNotImplemented = errors.New("journal mongo not implemented")
)

const (
	// JournalMongoCollection is the name of the collection in MongoDB
	JournalMongoCollection = "journal_entries"
)

// journalDocument represents a journal entry stored in MongoDB
type journalDocument struct {
	WorkflowID     string                   `bson:"workflowId"`
	Sequence       uint64                   `bson:"sequence"`
	Timestamp      int64                    `bson:"timestamp"`
	Type           string                   `bson:"type"`
	ThreadID       uint16                   `bson:"threadId"`
	FunctionNodeID string                   `bson:"functionNodeId,omitempty"`
	ExecID         string                   `bson:"execId,omitempty"`
	Input          map[string]any           `bson:"input,omitempty"`
	Result         *workflow.FunctionResult `bson:"result,omitempty"`
	State          string                   `bson:"state,omitempty"`
	ParentThreads  []uint16                 `bson:"parentThreads,omitempty"`
}

// MongoJournalRepository is a MongoDB implementation of the JournalRepository interface
type MongoJournalRepository struct {
	config     *config.Config
	client     *mongo.Client
	database   *mongo.Database
	collection *mongo.Collection
}

// NewMongoJournalRepository creates a new MongoDB JournalRepository
func NewMongoJournalRepository(client *mongo.Client, cfg *config.Config) JournalRepository {
	dbName := strutil.SerializeString(cfg.Database.Name)
	database := client.Database(dbName)
	collection := database.Collection(JournalMongoCollection)

	return &MongoJournalRepository{
		config:     cfg,
		client:     client,
		database:   database,
		collection: collection,
	}
}

// Append persists one or more journal entries for a workflow
func (m *MongoJournalRepository) Append(workflowID string, entries ...internalworkflow.JournalEntry) error {
	if len(entries) == 0 {
		return nil
	}

	docs := make([]any, 0, len(entries))
	for _, e := range entries {
		docs = append(docs, journalDocument{
			WorkflowID:     workflowID,
			Sequence:       e.Sequence,
			Timestamp:      e.Timestamp.Unix(),
			Type:           string(e.Type),
			ThreadID:       e.ThreadID,
			FunctionNodeID: e.FunctionNodeID,
			ExecID:         e.ExecID,
			Input:          e.Input,
			Result:         e.Result,
			State:          string(e.State),
			ParentThreads:  e.ParentThreads,
		})
	}

	_, err := m.collection.InsertMany(context.Background(), docs)
	return err
}

// LoadAll retrieves the full journal for a workflow, ordered by sequence
func (m *MongoJournalRepository) LoadAll(workflowID string) ([]internalworkflow.JournalEntry, error) {
	filter := bson.M{"workflowId": workflowID}
	opts := options.Find().SetSort(bson.D{{Key: "sequence", Value: 1}})

	cursor, err := m.collection.Find(context.Background(), filter, opts)
	if err != nil {
		return nil, err
	}
	defer func() { _ = cursor.Close(context.Background()) }()

	var docs []journalDocument
	if err := cursor.All(context.Background(), &docs); err != nil {
		return nil, err
	}

	entries := make([]internalworkflow.JournalEntry, 0, len(docs))
	for _, d := range docs {
		entries = append(entries, internalworkflow.JournalEntry{
			Sequence:       d.Sequence,
			Timestamp:      time.Unix(d.Timestamp, 0),
			Type:           internalworkflow.JournalEntryType(d.Type),
			ThreadID:       d.ThreadID,
			FunctionNodeID: d.FunctionNodeID,
			ExecID:         d.ExecID,
			Input:          d.Input,
			Result:         d.Result,
			State:          internalworkflow.State(d.State),
			ParentThreads:  d.ParentThreads,
		})
	}

	return entries, nil
}

// LastSequence returns the highest sequence number for a workflow
func (m *MongoJournalRepository) LastSequence(_ string) (uint64, error) {
	return 0, ErrJournalMongoNotImplemented
}
