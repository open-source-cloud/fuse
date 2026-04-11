package workflow

// TriggerType classifies how a workflow is initiated
type TriggerType string

const (
	// TriggerHTTP is the default trigger type — explicit API call
	TriggerHTTP TriggerType = "http"
	// TriggerCron triggers on a cron schedule
	TriggerCron TriggerType = "cron"
	// TriggerWebhook triggers when a specific webhook URL is called
	TriggerWebhook TriggerType = "webhook"
	// TriggerEvent triggers when a matching internal event is emitted
	TriggerEvent TriggerType = "event"
)

// TriggerConfig defines the trigger configuration for a workflow schema
type TriggerConfig struct {
	Type    TriggerType    `json:"type" validate:"required,oneof=http cron webhook event"`
	Cron    *CronConfig    `json:"cron,omitempty"`
	Webhook *WebhookConfig `json:"webhook,omitempty"`
	Event   *EventConfig   `json:"event,omitempty"`
}

// CronConfig defines cron trigger parameters
type CronConfig struct {
	// Expression is a cron expression (e.g., "0 */5 * * *" for every 5 minutes)
	Expression string `json:"expression" validate:"required"`
	// Timezone for cron evaluation (e.g., "America/New_York")
	Timezone string `json:"timezone,omitempty"`
	// Input is static input data passed to the trigger node on each execution
	Input map[string]any `json:"input,omitempty"`
}

// WebhookConfig defines webhook trigger parameters
type WebhookConfig struct {
	// Path is the custom webhook path (e.g., "/hooks/github-push")
	Path string `json:"path" validate:"required"`
	// Method is the HTTP method to listen for (default: POST)
	Method string `json:"method,omitempty"`
	// Secret is an optional HMAC secret for webhook signature verification
	Secret string `json:"secret,omitempty"`
}

// EventConfig defines event trigger parameters
type EventConfig struct {
	// EventType is the event name to listen for (e.g., "workflow.completed", "order.created")
	EventType string `json:"eventType" validate:"required"`
	// Filter is an optional expr-lang expression to filter matching events
	Filter string `json:"filter,omitempty"`
}
