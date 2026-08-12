package actors

// f01_2_verification_test.go — verification of BACKLOG_V2.md F-01, finding 2 ("f01-2-subworkflow"):
//
//   "Restart with a pending `system/subworkflow` spawns a second child — `findPendingThreads`
//    matches `step:started` without completion, `replayPendingThread` re-issues, interception
//    fires again."
//
// WHAT THIS TEST PROVES
//   - The OBSERVABLE defect only: after a restart of a run parked on a SYNCHRONOUS
//     system/subworkflow, MORE THAN ONE DISTINCT child workflow instance exists for the single
//     parent exec id that spawned it. Distinctness is collected from every place a child instance
//     becomes visible outside the handler: the SubWorkflowRef rows, the durable journal's
//     `subworkflow:started` entries, and the `workflow:trigger` messages the handler emitted.
//   - It deliberately asserts NOTHING about the route taken to get there: not `findPendingThreads`,
//     not the action `Resume()` returns, not which messages were sent, not how many repository rows
//     were written. Those are all things a legitimate fix may leave exactly as they are (see
//     "WHY THIS SHAPE" below), and asserting them would make this test red after a correct fix.
//
// WHAT THIS TEST DOES NOT PROVE
//   - Nothing here dies. No SIGKILL, no process restart, no ergo runtime, no supervision, no
//     mailbox ordering, no actor Init. The "restart" is modelled by rebuilding the aggregate from
//     the persisted journal and running a second WorkflowHandler value over the same repositories,
//     which is the shape of workflow_handler.go:150-159. Establishing that a real process death
//     and restart produces the same outcome is F-02's job ("kill with a pending
//     `system/subworkflow` (exactly one child after restart)"); until F-02 exists, this is engine
//     logic evidence, not crash-resume evidence.
//   - The handler is driven through a stub gen.Process (only Send and Log implemented), so the
//     child workflow is never actually started by WorkflowSupervisor. The test models that side
//     effect by inserting the child workflow row itself (see the Arrange block), because the
//     pre-crash world the backlog describes is one where the child IS already running.
//   - Memory repositories only. Postgres rebuilds the aggregate through
//     postgres/workflow.go:45 + loadGraph and stores refs in a table with its own constraints;
//     this test constructs the fresh aggregate by hand to match that shape but does not execute
//     the Postgres driver. Note MemoryWorkflowRepository.SaveSubWorkflowRef
//     (internal/repositories/workflow_memory.go:74-82) appends unconditionally with no dedup, which
//     is precisely why this test counts DISTINCT child ids and not rows.
//
// SCOPE — READ, NOT EXECUTED
//   The duplicate is asserted for a SYNCHRONOUS sub-workflow (async=false). Reading
//   handleSubWorkflowAction, the async=true branch calls SetResultFor at workflow_handler.go:936,
//   and SetResultFor (internal/workflow/workflow.go:421-441) appends a `step:completed` for that
//   exec id, so findPendingThreads (workflow.go:267) would never match it on replay. That path was
//   READ, not executed, and this file makes no claim about it.
//
// WHY THIS SHAPE (fix-agnosticism)
//   There are at least two legitimate fix sites, and the assertion below goes green under both:
//     (i)  the aggregate declines to re-issue — findPendingThreads/replayPendingThread
//          (workflow.go:267-306) or replayJournalEntries (workflow.go:199) learn to read
//          `subworkflow:started`, so Resume() no longer returns the sub-workflow step;
//     (ii) the actor stays idempotent — handleSubWorkflowAction (workflow_handler.go:901-902)
//          reuses the child id already journalled for that ParentExecID instead of minting a new
//          one with workflow.NewID(), and may still re-save the ref and re-send the trigger.
//   Under (ii) the aggregate is untouched, Resume() still returns system/subworkflow, a second ref
//   row is still written and a second trigger message is still sent — only the child IDENTITY
//   collapses to one. That is why the count is over distinct ids from all three sources.

import (
	"fmt"
	"sort"
	"testing"

	"ergo.services/ergo/gen"
	"github.com/open-source-cloud/fuse/internal/app/config"
	"github.com/open-source-cloud/fuse/internal/messaging"
	"github.com/open-source-cloud/fuse/internal/packages"
	"github.com/open-source-cloud/fuse/internal/packages/functions/system"
	"github.com/open-source-cloud/fuse/internal/repositories"
	internalworkflow "github.com/open-source-cloud/fuse/internal/workflow"
	"github.com/open-source-cloud/fuse/internal/workflow/workflowactions"
	"github.com/open-source-cloud/fuse/pkg/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	f012ParentSchemaID = "f01-2-parent"
	f012ChildSchemaID  = "f01-2-child-schema"
)

// f012Log is a no-op gen.Log. Only the levels the exercised code path uses are implemented; any
// other method would panic on the nil embedded interface, which is the intended loud failure.
type f012Log struct {
	gen.Log
}

func (l *f012Log) Trace(_ string, _ ...any)   {}
func (l *f012Log) Debug(_ string, _ ...any)   {}
func (l *f012Log) Info(_ string, _ ...any)    {}
func (l *f012Log) Warning(_ string, _ ...any) {}
func (l *f012Log) Error(_ string, _ ...any)   {}
func (l *f012Log) Panic(_ string, _ ...any)   {}

// f012Process is a stub gen.Process that records outbound sends instead of delivering them.
type f012Process struct {
	gen.Process
	log  *f012Log
	sent []f012Sent
}

type f012Sent struct {
	to      any
	message any
}

func (p *f012Process) Send(to any, message any) error {
	p.sent = append(p.sent, f012Sent{to: to, message: message})
	return nil
}

func (p *f012Process) Log() gen.Log { return p.log }

// f012ParentGraph builds n1 (ordinary function) -> n2 (system/subworkflow, schemaId mapped).
func f012ParentGraph(t *testing.T) *internalworkflow.Graph {
	t.Helper()
	schema := &internalworkflow.GraphSchema{
		ID:   f012ParentSchemaID,
		Name: "f01-2 parent workflow",
		Nodes: []*internalworkflow.NodeSchema{
			{ID: "n1", Function: "debug/nil"},
			{ID: "n2", Function: system.SubWorkflowFullFunctionID},
		},
		Edges: []*internalworkflow.EdgeSchema{
			{
				ID:   "e1",
				From: "n1",
				To:   "n2",
				Input: []internalworkflow.InputMapping{
					{Source: internalworkflow.SourceSchema, Value: f012ChildSchemaID, MapTo: "schemaId"},
				},
			},
		},
	}
	graph, err := internalworkflow.NewGraph(schema)
	require.NoError(t, err)
	// Same metadata shape as system.SubWorkflowFunctionMetadata(); normally installed by
	// GraphService.EnsureNodeMetadata (workflow_handler.go:139).
	require.NoError(t, graph.UpdateNodeMetadata("n2", &packages.FunctionMetadata{
		Input: packages.FunctionInputMetadata{
			Parameters: map[string]workflow.ParameterSchema{
				"schemaId": {Name: "schemaId", Type: "string", Required: true},
			},
		},
	}))
	return graph
}

// f012ChildGraph is a trivial two-node graph standing in for the child schema.
func f012ChildGraph(t *testing.T) *internalworkflow.Graph {
	t.Helper()
	graph, err := internalworkflow.NewGraph(&internalworkflow.GraphSchema{
		ID:   f012ChildSchemaID,
		Name: "f01-2 child workflow",
		Nodes: []*internalworkflow.NodeSchema{
			{ID: "c1", Function: "debug/nil"},
			{ID: "c2", Function: "debug/nil"},
		},
		Edges: []*internalworkflow.EdgeSchema{
			{ID: "ce1", From: "c1", To: "c2"},
		},
	})
	require.NoError(t, err)
	return graph
}

func f012NewHandler(
	wf *internalworkflow.Workflow,
	workflowRepo repositories.WorkflowRepository,
	journalRepo repositories.JournalRepository,
) (*WorkflowHandler, *f012Process) {
	handler := &WorkflowHandler{
		config:              &config.Config{},
		workflowRepository:  workflowRepo,
		journalRepo:         journalRepo,
		workflow:            wf,
		forEachStates:       make(map[string]*internalworkflow.ForEachState),
		iterThreadToForEach: make(map[uint16]string),
	}
	process := &f012Process{log: &f012Log{}}
	handler.Process = process
	return handler, process
}

// f012DistinctChildren collects the DISTINCT child workflow ids that exist for one parent exec id,
// from every observable a child instance can surface in, together with where each id was seen.
// It is deliberately tolerant: an id found in only one source still counts, and message shapes it
// does not recognise are ignored rather than failing, so a fix that changes the internal route
// cannot make this helper report a false duplicate.
func f012DistinctChildren(
	t *testing.T,
	workflowRepo repositories.WorkflowRepository,
	journalRepo repositories.JournalRepository,
	parentID string,
	parentExecID string,
	processes ...*f012Process,
) (childIDs []string, provenance []string) {
	t.Helper()

	seen := make(map[string][]string)

	refs, err := workflowRepo.FindActiveSubWorkflows(parentID)
	require.NoError(t, err)
	for _, ref := range refs {
		if ref.ParentExecID.String() != parentExecID {
			continue
		}
		id := ref.ChildWorkflowID.String()
		seen[id] = append(seen[id], "subWorkflowRef")
	}

	entries, err := journalRepo.LoadAll(parentID)
	require.NoError(t, err)
	for _, entry := range entries {
		if entry.Type != internalworkflow.JournalSubWorkflowStarted || entry.ExecID != parentExecID {
			continue
		}
		if id, ok := entry.Data["childWorkflowId"].(string); ok && id != "" {
			seen[id] = append(seen[id], fmt.Sprintf("journal:subworkflow:started#%d", entry.Sequence))
		}
	}

	for i, process := range processes {
		for _, sent := range process.sent {
			message, isMessage := sent.message.(messaging.Message)
			if !isMessage || message.Type != messaging.TriggerWorkflow {
				continue
			}
			trigger, isTrigger := message.Args.(messaging.TriggerWorkflowMessage)
			if !isTrigger || trigger.SchemaID != f012ChildSchemaID {
				continue
			}
			id := trigger.WorkflowID.String()
			seen[id] = append(seen[id], fmt.Sprintf("workflow:trigger(handler#%d)", i+1))
		}
	}

	childIDs = make([]string, 0, len(seen))
	for id := range seen {
		childIDs = append(childIDs, id)
	}
	sort.Strings(childIDs)

	provenance = make([]string, 0, len(childIDs))
	for _, id := range childIDs {
		provenance = append(provenance, fmt.Sprintf("%s seen in %v", id, seen[id]))
	}
	return childIDs, provenance
}

// The invariant under test: one synchronous system/subworkflow step execution owns exactly one
// child workflow instance, across any number of resumes.
func TestWorkflowHandler_ResumeWithPendingSubWorkflow_KeepsExactlyOneDistinctChild(t *testing.T) {
	// Arrange — run the parent up to the synchronous sub-workflow node and let the handler
	// intercept it once. This is the pre-crash world: one child spawned and in flight, parent
	// parked on it, journal persisted.
	workflowRepo := repositories.NewMemoryWorkflowRepository()
	journalRepo := repositories.NewMemoryJournalRepository()

	wfID := workflow.NewID()
	wf := internalworkflow.New(wfID, f012ParentGraph(t), "default")
	require.NoError(t, workflowRepo.Save(wf))

	triggerAction, ok := wf.Trigger().(*workflowactions.RunFunctionAction)
	require.True(t, ok, "Trigger must return a RunFunctionAction")
	wf.SetResultFor(triggerAction.FunctionExecID, &workflow.FunctionResult{
		Output: workflow.NewFunctionSuccessOutput(map[string]any{}),
	})
	subAction, ok := wf.Next(triggerAction.ThreadID).(*workflowactions.RunFunctionAction)
	require.True(t, ok, "Next must return a RunFunctionAction for n2")
	require.Equal(t, system.SubWorkflowFullFunctionID, subAction.FunctionID)
	require.Equal(t, f012ChildSchemaID, subAction.Args["schemaId"])
	parentExecID := subAction.FunctionExecID.String()

	preCrashHandler, preCrashProcess := f012NewHandler(wf, workflowRepo, journalRepo)
	preCrashHandler.handleWorkflowAction(subAction)

	preCrashChildren, preCrashProvenance := f012DistinctChildren(
		t, workflowRepo, journalRepo, wfID.String(), parentExecID, preCrashProcess)
	require.Lenf(t, preCrashChildren, 1,
		"precondition: the pre-crash run must own exactly one child (%v)", preCrashProvenance)
	require.Equal(t, internalworkflow.StateSleeping, wf.State(),
		"precondition: a synchronous sub-workflow parks the parent (workflow_handler.go:951)")

	// The child really is running when the parent dies: WorkflowSupervisor would have spawned a
	// handler for it, which Saves the child workflow (workflow_handler.go:172-175). The stub
	// process cannot do that, so record it here — otherwise a fix that legitimately respawns only
	// when the child never started would be judged against a world where no child exists.
	childID := workflow.ID(preCrashChildren[0])
	child := internalworkflow.New(childID, f012ChildGraph(t), "default")
	child.SetState(internalworkflow.StateRunning)
	require.NoError(t, workflowRepo.Save(child))

	// Act — restart: rebuild the aggregate from the persisted journal and hand Resume()'s action to
	// the handler, exactly as workflow_handler.go:150-159 does on recovery.
	entries, err := journalRepo.LoadAll(wfID.String())
	require.NoError(t, err)
	require.NotEmpty(t, entries, "the journal must have been persisted before the restart")

	resumed := internalworkflow.New(wfID, f012ParentGraph(t), "default")
	resumed.Journal().LoadFrom(entries)
	resumeHandler, resumeProcess := f012NewHandler(resumed, workflowRepo, journalRepo)
	resumeHandler.handleWorkflowAction(resumed.Resume())

	// Assert — how many distinct child workflows now exist for that one parent exec.
	childIDs, provenance := f012DistinctChildren(
		t, workflowRepo, journalRepo, wfID.String(), parentExecID, preCrashProcess, resumeProcess)
	assert.Lenf(t, childIDs, 1,
		"restart with a pending synchronous system/subworkflow produced %d distinct child workflows for parent exec %s; each of them is a full duplicate execution of schema %q:\n  %v",
		len(childIDs), parentExecID, f012ChildSchemaID, provenance)
}
