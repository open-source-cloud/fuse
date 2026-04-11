package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/open-source-cloud/fuse/internal/repositories"
	"github.com/open-source-cloud/fuse/internal/workflow"
	pkgwf "github.com/open-source-cloud/fuse/pkg/workflow"
	"golang.org/x/sync/errgroup"

	"github.com/open-source-cloud/fuse/pkg/objectstore"
)

// JournalRepository implements JournalRepository backed by PostgreSQL + ObjectStore.
type JournalRepository struct {
	pool  *pgxpool.Pool
	store objectstore.ObjectStore
}

// NewJournalRepository creates a new PostgreSQL-backed JournalRepository.
func NewJournalRepository(pool *pgxpool.Pool, store objectstore.ObjectStore) repositories.JournalRepository {
	return &JournalRepository{pool: pool, store: store}
}

func journalObjectKey(workflowID string, sequence uint64, suffix string) string {
	return fmt.Sprintf("workflows/%s/journal/%d/%s", workflowID, sequence, suffix)
}

// Append persists one or more journal entries for a workflow.
func (r *JournalRepository) Append(workflowID string, entries ...workflow.JournalEntry) error {
	ctx := context.Background()

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("postgres/journal: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	for _, entry := range entries {
		// Store payloads in object store
		var inputRef, resultRef, dataRef *string

		if entry.Input != nil {
			ref, err := r.putJSON(ctx, journalObjectKey(workflowID, entry.Sequence, "input.json"), entry.Input)
			if err != nil {
				return fmt.Errorf("postgres/journal: put input: %w", err)
			}
			inputRef = &ref
		}

		if entry.Result != nil {
			ref, err := r.putJSON(ctx, journalObjectKey(workflowID, entry.Sequence, "result.json"), entry.Result)
			if err != nil {
				return fmt.Errorf("postgres/journal: put result: %w", err)
			}
			resultRef = &ref
		}

		if entry.Data != nil {
			ref, err := r.putJSON(ctx, journalObjectKey(workflowID, entry.Sequence, "data.json"), entry.Data)
			if err != nil {
				return fmt.Errorf("postgres/journal: put data: %w", err)
			}
			dataRef = &ref
		}

		// Nullable state column
		var state *string
		if entry.State != "" {
			s := entry.State.String()
			state = &s
		}

		_, err = tx.Exec(ctx, `
			INSERT INTO journal_entries (
				workflow_id, sequence, entry_type, thread_id,
				function_node_id, exec_id, state, parent_threads,
				input_ref, result_ref, data_ref, created_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		`,
			workflowID, entry.Sequence, string(entry.Type), entry.ThreadID,
			nullIfEmpty(entry.FunctionNodeID), nullIfEmpty(entry.ExecID),
			state, int16Slice(entry.ParentThreads),
			inputRef, resultRef, dataRef, entry.Timestamp,
		)
		if err != nil {
			return fmt.Errorf("postgres/journal: insert entry seq=%d: %w", entry.Sequence, err)
		}
	}

	return tx.Commit(ctx)
}

// LoadAll retrieves the full journal for a workflow, ordered by sequence.
func (r *JournalRepository) LoadAll(workflowID string) ([]workflow.JournalEntry, error) {
	ctx := context.Background()

	rows, err := r.pool.Query(ctx, `
		SELECT sequence, entry_type, thread_id, function_node_id, exec_id,
		       state, parent_threads, input_ref, result_ref, data_ref, created_at
		FROM journal_entries
		WHERE workflow_id = $1
		ORDER BY sequence
	`, workflowID)
	if err != nil {
		return nil, fmt.Errorf("postgres/journal: load all: %w", err)
	}
	defer rows.Close()

	type entryRow struct {
		entry     workflow.JournalEntry
		inputRef  *string
		resultRef *string
		dataRef   *string
	}

	var entryRows []entryRow
	for rows.Next() {
		var er entryRow
		var entryType string
		var fnNodeID, execID, state *string
		var parentThreads []int16

		err := rows.Scan(
			&er.entry.Sequence, &entryType, &er.entry.ThreadID,
			&fnNodeID, &execID, &state, &parentThreads,
			&er.inputRef, &er.resultRef, &er.dataRef, &er.entry.Timestamp,
		)
		if err != nil {
			return nil, fmt.Errorf("postgres/journal: scan row: %w", err)
		}

		er.entry.Type = workflow.JournalEntryType(entryType)
		if fnNodeID != nil {
			er.entry.FunctionNodeID = *fnNodeID
		}
		if execID != nil {
			er.entry.ExecID = *execID
		}
		if state != nil {
			er.entry.State = workflow.State(*state)
		}
		er.entry.ParentThreads = uint16Slice(parentThreads)

		entryRows = append(entryRows, er)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres/journal: rows error: %w", err)
	}

	if len(entryRows) == 0 {
		return []workflow.JournalEntry{}, nil
	}

	// Batch-fetch S3 payloads in parallel with bounded concurrency to avoid
	// exhausting OS threads when recovering workflows with many journal entries.
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(10)
	entries := make([]workflow.JournalEntry, len(entryRows))

	for i := range entryRows {
		entries[i] = entryRows[i].entry
		er := entryRows[i]
		idx := i

		if er.inputRef != nil {
			ref := *er.inputRef
			g.Go(func() error {
				var m map[string]any
				if err := r.getJSON(gctx, ref, &m); err != nil {
					return fmt.Errorf("fetch input seq=%d: %w", entries[idx].Sequence, err)
				}
				entries[idx].Input = m
				return nil
			})
		}

		if er.resultRef != nil {
			ref := *er.resultRef
			g.Go(func() error {
				var fr pkgwf.FunctionResult
				if err := r.getJSON(gctx, ref, &fr); err != nil {
					return fmt.Errorf("fetch result seq=%d: %w", entries[idx].Sequence, err)
				}
				entries[idx].Result = &fr
				return nil
			})
		}

		if er.dataRef != nil {
			ref := *er.dataRef
			g.Go(func() error {
				var m map[string]any
				if err := r.getJSON(gctx, ref, &m); err != nil {
					return fmt.Errorf("fetch data seq=%d: %w", entries[idx].Sequence, err)
				}
				entries[idx].Data = m
				return nil
			})
		}
	}

	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("postgres/journal: fetch payloads: %w", err)
	}

	return entries, nil
}

// LastSequence returns the highest sequence number for a workflow.
func (r *JournalRepository) LastSequence(workflowID string) (uint64, error) {
	ctx := context.Background()

	var seq *int64
	err := r.pool.QueryRow(ctx,
		`SELECT MAX(sequence) FROM journal_entries WHERE workflow_id = $1`, workflowID,
	).Scan(&seq)
	if err != nil {
		return 0, fmt.Errorf("postgres/journal: last sequence: %w", err)
	}
	if seq == nil {
		return 0, nil
	}
	return safeInt64ToUint64(*seq), nil
}

// putJSON marshals v to JSON and stores it in the object store, returning the key.
func (r *JournalRepository) putJSON(ctx context.Context, key string, v any) (string, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("marshal: %w", err)
	}
	if err := r.store.Put(ctx, key, data); err != nil {
		return "", fmt.Errorf("put %q: %w", key, err)
	}
	return key, nil
}

// getJSON fetches a JSON object from the store and unmarshals it into dst.
func (r *JournalRepository) getJSON(ctx context.Context, key string, dst any) error {
	data, err := r.store.Get(ctx, key)
	if err != nil {
		return fmt.Errorf("get %q: %w", key, err)
	}
	return json.Unmarshal(data, dst)
}

// FindFailed returns all step:failed journal entries for the given workflow.
func (r *JournalRepository) FindFailed(workflowID string) ([]workflow.JournalEntry, error) {
	ctx := context.Background()
	rows, err := r.pool.Query(ctx, `
		SELECT sequence, entry_type::TEXT, thread_id, function_node_id, exec_id, created_at
		FROM journal_entries
		WHERE workflow_id = $1 AND entry_type = 'step:failed'
		ORDER BY sequence
	`, workflowID)
	if err != nil {
		return nil, fmt.Errorf("postgres/journal: find failed: %w", err)
	}
	defer rows.Close()

	var entries []workflow.JournalEntry
	for rows.Next() {
		var e workflow.JournalEntry
		var seq int64
		var threadID int16
		var nodeID, execID *string
		if scanErr := rows.Scan(&seq, &e.Type, &threadID, &nodeID, &execID, &e.Timestamp); scanErr != nil {
			return nil, fmt.Errorf("postgres/journal: scan failed entry: %w", scanErr)
		}
		e.Sequence = safeInt64ToUint64(seq)
		e.ThreadID = safeInt16ToUint16(threadID)
		if nodeID != nil {
			e.FunctionNodeID = *nodeID
		}
		if execID != nil {
			e.ExecID = *execID
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// nullIfEmpty returns nil for empty strings, for nullable VARCHAR columns.
func nullIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// safeUint16ToInt16 converts uint16 to int16 safely.
// Thread IDs in FUSE are uint16 (0–65535); PostgreSQL SMALLINT is int16 (-32768–32767).
// Values above 32767 are stored as their two's-complement int16 representation
// and round-trip correctly through safeInt16ToUint16.
func safeUint16ToInt16(v uint16) int16 {
	return int16(v) //nolint:gosec // intentional two's-complement for SMALLINT storage
}

// safeInt16ToUint16 converts int16 back to uint16.
func safeInt16ToUint16(v int16) uint16 {
	return uint16(v) //nolint:gosec // inverse of safeUint16ToInt16
}

// safeInt64ToUint64 converts int64 to uint64.
// Journal sequences are always non-negative so this is safe.
func safeInt64ToUint64(v int64) uint64 {
	return uint64(v) //nolint:gosec // sequences are always non-negative
}

// int16Slice converts []uint16 to []int16 for PostgreSQL SMALLINT[] compatibility.
func int16Slice(u []uint16) []int16 {
	if u == nil {
		return nil
	}
	s := make([]int16, len(u))
	for i, v := range u {
		s[i] = safeUint16ToInt16(v)
	}
	return s
}

// uint16Slice converts []int16 back to []uint16.
func uint16Slice(s []int16) []uint16 {
	if s == nil {
		return nil
	}
	u := make([]uint16, len(s))
	for i, v := range s {
		u[i] = safeInt16ToUint16(v)
	}
	return u
}
