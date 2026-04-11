package handlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"ergo.services/ergo/gen"
	"github.com/open-source-cloud/fuse/internal/dtos"
	"github.com/open-source-cloud/fuse/internal/messaging"
	"github.com/open-source-cloud/fuse/internal/services"
	internalworkflow "github.com/open-source-cloud/fuse/internal/workflow"
	"github.com/open-source-cloud/fuse/pkg/workflow"
)

type (
	// WebhookHandler handles incoming webhook requests and triggers matching workflows
	WebhookHandler struct {
		Handler
		graphService services.GraphService
	}
	// WebhookHandlerFactory is a factory for creating WebhookHandler actors
	WebhookHandlerFactory HandlerFactory[*WebhookHandler]
)

const (
	// WebhookHandlerName is the name of the WebhookHandler actor
	WebhookHandlerName = "webhook_handler"
	// WebhookHandlerPoolName is the name of the WebhookHandler pool
	WebhookHandlerPoolName = "webhook_handler_pool"
)

// NewWebhookHandlerFactory creates a new WebhookHandlerFactory
func NewWebhookHandlerFactory(graphService services.GraphService) *WebhookHandlerFactory {
	return &WebhookHandlerFactory{
		Factory: func() gen.ProcessBehavior {
			return &WebhookHandler{
				graphService: graphService,
			}
		},
	}
}

// HandlePost handles incoming webhook requests (POST /v1/hooks/{path:.*})
// @Summary Handle incoming webhook
// @Description Routes incoming webhooks to the matching workflow trigger
// @Tags webhooks
// @Accept json
// @Produce json
// @Param path path string true "Webhook path"
// @Success 200 {object} dtos.TriggerWorkflowResponse
// @Failure 400 {object} dtos.BadRequestError
// @Failure 404 {object} dtos.NotFoundError
// @Failure 500 {object} dtos.InternalServerErrorResponse
// @Router /v1/hooks/{path} [post]
func (h *WebhookHandler) HandlePost(_ gen.PID, w http.ResponseWriter, r *http.Request) error {
	webhookPath, err := h.GetPathParam(r, "path")
	if err != nil {
		return h.SendBadRequest(w, err, EmptyFields)
	}
	webhookPath = "/" + webhookPath

	// Find matching schema by scanning all schemas with webhook triggers
	schemaID, webhookCfg, err := h.resolveWebhook(webhookPath)
	if err != nil {
		return h.SendNotFound(w, fmt.Sprintf("no webhook registered for path %s", webhookPath), EmptyFields)
	}

	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return h.SendBadRequest(w, fmt.Errorf("failed to read request body"), EmptyFields)
	}

	// Verify HMAC signature if configured
	if webhookCfg.Secret != "" {
		signature := r.Header.Get("X-Hub-Signature-256")
		if !verifyHMAC(body, signature, webhookCfg.Secret) {
			return h.SendBadRequest(w, fmt.Errorf("invalid webhook signature"), []string{"X-Hub-Signature-256"})
		}
	}

	// Parse body as input data
	var input map[string]any
	if len(body) > 0 {
		if bindErr := h.BindJSONBytes(body, &input); bindErr != nil {
			// If not JSON, pass raw body as "body" field
			input = map[string]any{"body": string(body)}
		}
	}

	workflowID := workflow.NewID()
	triggerMsg := messaging.NewTriggerWorkflowWithInputMessage(schemaID, workflowID, input)
	if sendErr := h.Send(WorkflowSupervisorName, triggerMsg); sendErr != nil {
		return h.SendInternalError(w, sendErr)
	}

	return h.SendJSON(w, http.StatusOK, dtos.TriggerWorkflowResponse{
		SchemaID:   schemaID,
		WorkflowID: workflowID.String(),
		Code:       "OK",
	})
}

func (h *WebhookHandler) resolveWebhook(path string) (string, *internalworkflow.WebhookConfig, error) {
	schemas, err := h.graphService.ListSchemas()
	if err != nil {
		return "", nil, err
	}

	for _, item := range schemas {
		graph, gErr := h.graphService.FindByID(item.SchemaID)
		if gErr != nil {
			continue
		}
		tc := graph.Schema().TriggerConfig
		if tc == nil || tc.Type != internalworkflow.TriggerWebhook || tc.Webhook == nil {
			continue
		}
		if tc.Webhook.Path == path {
			return graph.Schema().ID, tc.Webhook, nil
		}
	}

	return "", nil, fmt.Errorf("no webhook for path %s", path)
}

// BindJSONBytes decodes JSON from raw bytes
func (h *WebhookHandler) BindJSONBytes(data []byte, v any) error {
	return json.Unmarshal(data, v)
}

func verifyHMAC(body []byte, signature, secret string) bool {
	if signature == "" {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expected := "sha256=" + hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}
