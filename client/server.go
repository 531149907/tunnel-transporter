package client

import (
	"fmt"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"net"
	"tunnel-transporter/config"
	"tunnel-transporter/constants"
	"tunnel-transporter/message"
	"tunnel-transporter/proxy"
	"tunnel-transporter/registry"
	"tunnel-transporter/util"
)

var (
	proxyRegistry = registry.NewRegistryManager()
)

func StartServer() {
	listener, err := util.Listen(int(config.ClientConfig.Server.Port))
	defer listener.Close()

	if err != nil {
		_, port := util.ResolveAddress(listener.Addr().String())
		log.Panicf("error while listening on %d, reason: %v", port, err)
		return
	}

	for {
		conn, err := listener.AcceptTCP()
		if err != nil {
			log.Errorf("error while accepting connection, reason: %v", err)
		}

		go handleAgentConnection(conn)
	}
}

func handleAgentConnection(conn *net.TCPConn) {
	firstMessage, err := util.Read(conn)
	if err != nil || firstMessage == nil {
		log.Errorf("error reading message from connection, reason: %v", err)
		return
	}

	switch firstMessage.GetType() {
	case message.BootstrapRequest:
		if err := handleBootstrapConnection(*firstMessage.(*message.BootstrapRequestMessage), conn); err != nil {
			log.Errorf("error handling bootstrap connection, reason: %v", err)
		}
	case message.RequireConnectionResponse:
		handleNewConnection(*firstMessage.(*message.RequireNewConnectionResponseMessage), conn)
	default:
		log.Warn("received unknown message")
	}
}

func handleBootstrapConnection(requestMessage message.BootstrapRequestMessage, conn *net.TCPConn) error {
	if config.ClientConfig.Server.Authentication.Type == constants.StaticToken {
		if requestMessage.StaticToken != config.ClientConfig.Server.Authentication.StaticToken.Token {
			_ = util.Write(conn, message.BootstrapResponseMessage{Error: "invalid token"})
			conn.Close()
			return errors.New(fmt.Sprintf("agent %s bootstrap with invalid token", requestMessage.AgentId))
		}
	}

	tunnel, err := proxy.NewProxy(requestMessage.AgentId, requestMessage.AgentVersion, conn, proxyRegistry.UnregisterChan)
	if err != nil {
		log.Errorf("error creating new tunnel, reason: %v", err)
		return err
	}

	proxyRegistry.Put(tunnel.PublicListenPort, tunnel)
	return nil
}

func handleNewConnection(responseMessage message.RequireNewConnectionResponseMessage, conn *net.TCPConn) {
	if tunnelProxy := proxyRegistry.GetByAgentId(responseMessage.AgentId); tunnelProxy == nil {
		log.Warnf("fail to find tunnel proxy for agent %s", responseMessage.AgentId)
	} else {
		tunnelProxy.HandleNewDataConnection(responseMessage, conn)
	}
}
