package signal

import (
	"encoding/json"
)

type Type string

type rawSignal struct {
	Type    Type
	Payload json.RawMessage
}

type TypedSignal interface {
	GetType() Type
}
