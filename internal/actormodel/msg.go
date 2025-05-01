// Package actormodel actor model helpers
package actormodel

// MessageType actor message type
type MessageType string

// MessageData actor message data type
type MessageData map[string]any

// Message defines a message interface
type Message interface {
	Type() MessageType
	Data() MessageData
}

// Str gets a string from message data
func (d MessageData) Str(key string) string {
	val, ok := d[key]
	if !ok {
		return ""
	}
	return val.(string)
}

type message struct {
	msgType MessageType
	data    MessageData
}

// NewMessage returns a new Message object with type and data
func NewMessage(msgType MessageType, data MessageData) Message {
	return &message{
		msgType: msgType,
		data:    data,
	}
}

func (m *message) Type() MessageType {
	return m.msgType
}

func (m *message) Data() MessageData {
	return m.data
}
