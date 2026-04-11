package events

// EventBus manages event subscriptions and publishing
type EventBus interface {
	// Publish emits an event to all matching subscribers
	Publish(event Event) error
	// Subscribe registers a callback for events matching the given type
	Subscribe(eventType string, handler EventHandler) (SubscriptionID, error)
	// Unsubscribe removes a subscription
	Unsubscribe(id SubscriptionID) error
	// Close stops the event bus and cleans up resources
	Close() error
}
