package server

import "github.com/open-source-cloud/fuse/internal/actormodel"

type message struct {
	msgType actormodel.MessageType
}

// NewMessage returns a new Message object with type and data
func NewMessage(msgType actormodel.MessageType) actormodel.Message {
	return &message{
		msgType: msgType,
	}
}

func (m *message) Type() actormodel.MessageType {
	return m.msgType
}

func (m *message) Data() actormodel.MessageData {
	return nil
}
