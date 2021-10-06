package signal

import (
	"encoding/json"
)

const (
	Ping          Type = "Ping"
	Pong          Type = "Pong"
	BootstrapReq  Type = "BootstrapReq"
	BootstrapResp Type = "BootstrapResp"
	NewConnReq    Type = "NewConnReq"
	NewConnResp   Type = "NewConnResp"
)

func Pack(payload TypedSignal) ([]byte, error) {
	return json.Marshal(struct {
		Type    Type
		Payload interface{}
	}{
		Type:    payload.GetType(),
		Payload: payload,
	})
}

func Unpack(buffer []byte) (TypedSignal, error) {
	var raw rawSignal
	if err := json.Unmarshal(buffer, &raw); err != nil {
		return nil, err
	}

	var signal TypedSignal
	switch raw.Type {
	case Ping:
		signal = &PingSignal{}
	case Pong:
		signal = &PongSignal{}
	case BootstrapReq:
		signal = &BootstrapReqSignal{}
	case BootstrapResp:
		signal = &BootstrapRespSignal{}
	case NewConnReq:
		signal = &NewConnReqSignal{}
	case NewConnResp:
		signal = &NewConnRespSignal{}
	}

	if err := json.Unmarshal(raw.Payload, &signal); err != nil {
		return nil, err
	}

	return signal, nil
}

/*===Ping===*/

type PingSignal struct {
}

func (p PingSignal) GetType() Type {
	return Ping
}

/*===Pong===*/

type PongSignal struct {
}

func (p PongSignal) GetType() Type {
	return Pong
}

/*===BootstrapReq===*/

type BootstrapReqSignal struct {
	AgentVersion string
	AgentId      string
	OS           string
	Arch         string

	StaticToken string
}

func (b BootstrapReqSignal) GetType() Type {
	return BootstrapReq
}

/*===BootstrapResp===*/

type BootstrapRespSignal struct {
	Error string
}

func (b BootstrapRespSignal) GetType() Type {
	return BootstrapResp
}

/*===NewConnReq===*/

type NewConnReqSignal struct {
	RequestId string
	HttpConn  bool
}

func (r NewConnReqSignal) GetType() Type {
	return NewConnReq
}

/*===NewConnResp===*/

type NewConnRespSignal struct {
	AgentId     string
	StaticToken string
	Error       string
	RequestId   string
	HttpConn    bool
}

func (r NewConnRespSignal) GetType() Type {
	return NewConnResp
}
