package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"time"

	"github.com/open-source-cloud/fuse/internal/app/config"
	"github.com/open-source-cloud/fuse/internal/app/di"
	"github.com/open-source-cloud/fuse/internal/logging"
	"github.com/open-source-cloud/fuse/internal/services"
	"github.com/open-source-cloud/fuse/internal/workflow"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"go.uber.org/fx"
)

// workflowSpecFile is the path to the workflow spec file (-c/--config).
var workflowSpecFile string

// workflowEnvironment scopes secret resolution for the run (-e/--environment); empty uses the
// engine default (FUSE_ENVIRONMENT).
var workflowEnvironment string

const (
	workflowRunHealthAttempts = 60
	workflowRunPollInterval   = 250 * time.Millisecond
	workflowRunTimeout        = 5 * time.Minute
	workflowRunHTTPTimeout    = 30 * time.Second
)

func newWorkflowCommand() *cobra.Command {
	workflowCmd := &cobra.Command{
		Use:   "workflow",
		Short: "Run a workflow once, in-memory, and print its result",
		Long: "Loads a workflow graph JSON, runs it once entirely in-memory (no external server needed), " +
			"waits for it to finish, prints the execution snapshot, and exits non-zero if it errored. " +
			"Configure AI providers via env (e.g. LLM_OLLAMA_*) to run ai/chat and ai/agent workflows.",
		Args: cobra.NoArgs,
		RunE: func(*cobra.Command, []string) error { return runWorkflowApp() },
	}
	workflowCmd.Flags().StringVarP(&workflowSpecFile, "config", "c", "", "Path to the workflow spec JSON file")
	workflowCmd.Flags().StringVarP(&workflowEnvironment, "environment", "e", "", "Environment scope for secret resolution (defaults to FUSE_ENVIRONMENT)")
	return workflowCmd
}

// runWorkflowApp boots the full app in-memory, runs the workflow once, and exits.
func runWorkflowApp() error {
	if workflowSpecFile == "" {
		return errors.New("a workflow spec file is required (-c <file.json>)")
	}
	if path.Ext(workflowSpecFile) != ".json" {
		return errors.New("only .json workflow spec files are supported")
	}
	spec, err := os.ReadFile(workflowSpecFile) //nolint:gosec // CLI reads a user-provided path
	if err != nil {
		return fmt.Errorf("read workflow spec: %w", err)
	}

	resultCh := make(chan error, 1)
	app := fx.New(
		di.AllModules,
		fx.Invoke(func(cfg *config.Config, gs services.GraphService, lc fx.Lifecycle, sd fx.Shutdowner) {
			lc.Append(fx.Hook{
				OnStart: func(context.Context) error {
					// Run the workflow off the lifecycle goroutine so the server's own
					// OnStart hooks can finish; then shut the app down so Run() returns.
					go func() {
						resultCh <- runWorkflowOnce(cfg, gs, spec)
						_ = sd.Shutdown()
					}()
					return nil
				},
			})
		}),
		fx.WithLogger(logging.NewFxLogger()),
	)

	app.Run()
	if err := app.Err(); err != nil {
		return err
	}
	select {
	case err := <-resultCh:
		return err
	default:
		return nil
	}
}

// runWorkflowOnce upserts the graph, triggers it against the in-process server,
// waits for a terminal state, prints the snapshot, and returns an error if the
// workflow did not finish successfully.
func runWorkflowOnce(cfg *config.Config, gs services.GraphService, spec []byte) error {
	base := "http://localhost:" + cfg.Server.Port
	client := &http.Client{Timeout: workflowRunHTTPTimeout}

	if err := workflowWaitHealth(client, base); err != nil {
		return err
	}

	schema, err := workflow.NewGraphSchemaFromJSON(spec)
	if err != nil {
		return fmt.Errorf("parse workflow spec: %w", err)
	}
	graph, err := gs.Upsert(schema.ID, schema)
	if err != nil {
		return fmt.Errorf("upsert workflow graph: %w", err)
	}

	wfID, err := workflowTrigger(client, base, graph.ID(), workflowEnvironment)
	if err != nil {
		return err
	}
	log.Info().Str("schemaID", graph.ID()).Str("workflowID", wfID).Msg("workflow triggered; waiting for completion")

	status, err := workflowWaitTerminal(client, base, wfID)
	if err != nil {
		return err
	}

	printWorkflowResult(client, base, wfID, status)

	if status != "finished" {
		return fmt.Errorf("workflow %s ended in status %q", wfID, status)
	}
	return nil
}

func workflowWaitHealth(client *http.Client, base string) error {
	for i := 0; i < workflowRunHealthAttempts; i++ {
		resp, err := client.Get(base + "/health") //nolint:noctx // short-lived CLI poll
		if err == nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		time.Sleep(workflowRunPollInterval)
	}
	return errors.New("workflow runner: in-process server did not become healthy")
}

func workflowTrigger(client *http.Client, base, schemaID, environment string) (string, error) {
	reqBody := map[string]string{"schemaID": schemaID}
	if environment != "" {
		reqBody["environment"] = environment
	}
	payload, _ := json.Marshal(reqBody)
	resp, err := client.Post(base+"/v1/workflows/trigger", "application/json", bytes.NewReader(payload)) //nolint:noctx // short-lived CLI call
	if err != nil {
		return "", fmt.Errorf("trigger workflow: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return "", fmt.Errorf("trigger workflow: status %d: %s", resp.StatusCode, string(body))
	}
	var tr struct {
		WorkflowID string `json:"workflowId"`
	}
	if err := json.Unmarshal(body, &tr); err != nil || tr.WorkflowID == "" {
		return "", fmt.Errorf("trigger workflow: unexpected response: %s", string(body))
	}
	return tr.WorkflowID, nil
}

func workflowWaitTerminal(client *http.Client, base, wfID string) (string, error) {
	deadline := time.Now().Add(workflowRunTimeout)
	url := base + "/v1/workflows/" + wfID
	for time.Now().Before(deadline) {
		if status, ok := workflowFetchStatus(client, url); ok && isTerminalWorkflowStatus(status) {
			return status, nil
		}
		time.Sleep(workflowRunPollInterval)
	}
	return "", fmt.Errorf("workflow %s did not reach a terminal state within %s", wfID, workflowRunTimeout)
}

func workflowFetchStatus(client *http.Client, url string) (string, bool) {
	resp, err := client.Get(url) //nolint:noctx // short-lived CLI poll
	if err != nil {
		return "", false
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", false
	}
	var s struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(body, &s); err != nil {
		return "", false
	}
	return s.Status, true
}

func isTerminalWorkflowStatus(status string) bool {
	switch status {
	case "finished", "error", "cancelled":
		return true
	default:
		return false
	}
}

// printWorkflowResult prints the execution snapshot (node outputs) to stdout.
func printWorkflowResult(client *http.Client, base, wfID, status string) {
	resp, err := client.Get(base + "/v1/workflows/" + wfID + "/snapshot") //nolint:noctx // short-lived CLI call
	if err != nil {
		log.Warn().Err(err).Msg("could not fetch workflow snapshot")
		return
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(resp.Body)
	var pretty bytes.Buffer
	if json.Indent(&pretty, body, "", "  ") == nil {
		body = pretty.Bytes()
	}
	fmt.Printf("\nworkflow %s status=%s\nresult snapshot:\n%s\n", wfID, status, string(body))
}
