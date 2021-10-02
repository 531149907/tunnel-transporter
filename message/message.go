package message

import (
	"encoding/json"
	"github.com/pkg/errors"
)

type (
	Type    string
	Content interface{}
)

const (
	Unknown                   Type = "Unknown"
	Ping                      Type = "Ping"
	Pong                      Type = "Pong"
	BootstrapRequest          Type = "BootstrapRequest"
	BootstrapResponse         Type = "BootstrapResponse"
	RequireConnectionRequest  Type = "RequireConnectionRequest"
	RequireConnectionResponse Type = "RequireConnectionResponse"
)

type RawMessage struct {
	Type    Type
	Payload json.RawMessage
}

type TypedMessage interface {
	GetType() Type
}

func Pack(payload TypedMessage) ([]byte, error) {
	return json.Marshal(struct {
		Type    Type
		Payload Content
	}{
		Type:    payload.GetType(),
		Payload: payload,
	})
}

func Unpack(buffer []byte) (TypedMessage, error) {
	var rawMessage RawMessage
	err := json.Unmarshal(buffer, &rawMessage)
	if err != nil {
		return nil, err
	}

	var message TypedMessage
	switch rawMessage.Type {
	case Ping:
		message = &PingMessage{}
	case Pong:
		message = &PongMessage{}
	case BootstrapRequest:
		message = &BootstrapRequestMessage{}
	case BootstrapResponse:
		message = &BootstrapResponseMessage{}
	case RequireConnectionRequest:
		message = &RequireNewConnectionRequestMessage{}
	case RequireConnectionResponse:
		message = &RequireNewConnectionResponseMessage{}
	default:
		return nil, errors.New("unknown message type")
	}

	err = json.Unmarshal(rawMessage.Payload, &message)
	if err != nil {
		return nil, err
	}

	return message, nil
}

/*===Ping===*/

type PingMessage struct {
}

func (p PingMessage) GetType() Type {
	return Ping
}

/*===Pong===*/

type PongMessage struct {
}

func (p PongMessage) GetType() Type {
	return Pong
}

/*===BootstrapRequest===*/

type BootstrapRequestMessage struct {
	AgentVersion string
	AgentId      string
	OS           string
	Arch         string

	StaticToken string
}

func (b BootstrapRequestMessage) GetType() Type {
	return BootstrapRequest
}

/*===BootstrapResponse===*/

type BootstrapResponseMessage struct {
	Error string
}

func (b BootstrapResponseMessage) GetType() Type {
	return BootstrapResponse
}

/*===RequireConnectionRequest===*/

type RequireNewConnectionRequestMessage struct {
}

func (r RequireNewConnectionRequestMessage) GetType() Type {
	return RequireConnectionRequest
}

/*===RequireConnectionResponse===*/

type RequireNewConnectionResponseMessage struct {
	AgentId string

	StaticToken string

	Error string
}

func (r RequireNewConnectionResponseMessage) GetType() Type {
	return RequireConnectionResponse
}
