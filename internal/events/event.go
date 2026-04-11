// Package events provides an internal event bus for workflow lifecycle events
package events

import "time"

// Event represents an internal system event
type Event struct {
	Type      string         `json:"type"`
	Source    string         `json:"source"`
	Timestamp time.Time      `json:"timestamp"`
	Data      map[string]any `json:"data"`
}

// EventHandler is a callback function invoked when a matching event is published
type EventHandler func(event Event) error

// SubscriptionID uniquely identifies a subscription
type SubscriptionID string
