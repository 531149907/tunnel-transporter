package proxy

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"net"
	"time"
	"tunnel-transporter/config"
	"tunnel-transporter/message"
	"tunnel-transporter/util"
)

type BootstrapConnection struct {
	raw *RawConnection

	pingTicker        *time.Ticker
	pingTimeoutTicker *time.Ticker
	pongTimeoutTicker *time.Ticker
}

func NewBootstrapConnection(ctx context.Context, cancel chan<- error, conn *net.TCPConn, isServer bool) *BootstrapConnection {
	bootstrap := BootstrapConnection{
		raw:               NewRawConnection(ctx, cancel, conn),
		pingTicker:        time.NewTicker(10 * time.Second),
		pingTimeoutTicker: time.NewTicker(30 * time.Second),
		pongTimeoutTicker: time.NewTicker(30 * time.Second),
	}

	if isServer {
		bootstrap.pingTicker.Stop()
		bootstrap.pongTimeoutTicker.Stop()
	} else {
		bootstrap.pingTimeoutTicker.Stop()
		bootstrap.raw.write(message.BootstrapRequestMessage{
			AgentId:     config.ClientConfig.Agent.Id,
			StaticToken: config.ClientConfig.Agent.Authentication.StaticToken.Token,
		})
	}

	go bootstrap.shutdown(ctx)
	go bootstrap.handleCommand(ctx, cancel)

	return &bootstrap
}

func (b *BootstrapConnection) handleCommand(ctx context.Context, cancel chan<- error) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-b.pingTicker.C:
			b.raw.write(message.PingMessage{})
			b.pongTimeoutTicker.Reset(10 * time.Second)
		case <-b.pingTimeoutTicker.C:
			err := errors.New("heartbeat failure, ping did not receive in time")
			log.Error(err)
			cancel <- err
			return
		case <-b.pongTimeoutTicker.C:
			err := errors.New("heartbeat failure, pong did not receive in time")
			log.Error(err)
			cancel <- err
			return
		default:
			receivedMessage := b.raw.read()
			if receivedMessage == nil {
				continue
			}

			log.Debugf("receive command %s", receivedMessage.GetType())

			go func() {
				switch receivedMessage.GetType() {
				case message.Ping:
					b.handlePing(*receivedMessage.(*message.PingMessage))
				case message.Pong:
					b.handlePong(*receivedMessage.(*message.PongMessage))
				case message.RequireConnectionRequest:
					b.handleRequireConnectionRequest(ctx, cancel, *receivedMessage.(*message.RequireNewConnectionRequestMessage))
				case message.BootstrapResponse:
					b.handleBootstrapResponse(ctx, cancel, *receivedMessage.(*message.BootstrapResponseMessage))
				case message.BootstrapRequest, message.RequireConnectionResponse:
					//no need to implement
				default:
					log.Warn("received unknown message type")
				}
			}()
		}
	}
}

func (b *BootstrapConnection) handlePing(pingMessage message.PingMessage) {
	b.raw.write(message.PongMessage{})
	b.pingTimeoutTicker.Stop()
}

func (b *BootstrapConnection) handlePong(pongMessage message.PongMessage) {
	b.pongTimeoutTicker.Stop()
}

func (b *BootstrapConnection) handleRequireConnectionRequest(ctx context.Context, cancel chan<- error, requestMessage message.RequireNewConnectionRequestMessage) {
	localIp, localPort := util.ResolveAddress(config.ClientConfig.Agent.LocalEndpoint)
	localConnection, err := util.Dial(localIp, localPort)
	if err != nil {
		log.Errorf("error dialing local service, reason: %v", err)
		return
	}

	serverIp, serverPort := util.ResolveAddress(config.ClientConfig.Agent.ServerEndpoint)
	proxyConnection, err := util.Dial(serverIp, serverPort)
	if err != nil {
		log.Errorf("error crearing proxy connection, reason: %v", err)
		return
	}

	wrappedProxyConnection := NewDataConnection(ctx, cancel, proxyConnection)
	err = util.Write(proxyConnection, message.RequireNewConnectionResponseMessage{
		AgentId:     config.ClientConfig.Agent.Id,
		StaticToken: config.ClientConfig.Agent.Authentication.StaticToken.Token})
	if err != nil {
		log.Errorf("error writing connection, reason: %v", err)
		return
	}

	wrappedProxyConnection.join(localConnection)
}

func (b *BootstrapConnection) handleBootstrapResponse(ctx context.Context, cancel chan<- error, responseMessage message.BootstrapResponseMessage) {
	defer func() {
		if err := recover(); err != nil {
			log.Warnf("error handling bootstrapResponse message, reason: %v", err)
		}
	}()

	if responseMessage.Error != "" {
		cancel <- errors.New(fmt.Sprintf("error creating bootstrap connection, reason, %v", responseMessage.Error))
		return
	}
}

func (b *BootstrapConnection) shutdown(ctx context.Context) {
	select {
	case <-ctx.Done():
	}
}
