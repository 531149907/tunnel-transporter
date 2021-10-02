package message

import (
	"fmt"
	"testing"
)

func TestPack(t *testing.T) {
	bootstrapRequestMessage := BootstrapRequestMessage{
		AgentVersion: "1.0",
		AgentId:      "ABC",
	}

	pack, err := Pack(bootstrapRequestMessage)
	if err != nil {
		return
	}

	fmt.Println(string(pack))
}

func TestUnpack(t *testing.T) {
	bootstrapRequestMessage := BootstrapRequestMessage{
		AgentVersion: "1.0",
		AgentId:      "ABC",
	}
	packedBytes, err := Pack(bootstrapRequestMessage)
	if err != nil {
		return
	}

	recoveredMessage, err := Unpack(packedBytes)
	if err != nil {
		return
	}

	fmt.Println(recoveredMessage)
}
