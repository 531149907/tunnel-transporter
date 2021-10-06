package connection

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"net"
	"time"
	"tunnel-transporter/internal/config"
	"tunnel-transporter/internal/signal"
	"tunnel-transporter/internal/util"
)

type CtrlConn struct {
	*rawConn

	cancelChan chan<- error

	pingTicker        *time.Ticker
	pingTimeoutTicker *time.Ticker
	pongTimeoutTicker *time.Ticker
}

func NewCtrl(ctx context.Context, cancel chan<- error, conn net.Conn, isServer bool) *CtrlConn {
	bootstrap := CtrlConn{
		rawConn:           wrapRaw(conn),
		cancelChan:        cancel,
		pingTicker:        time.NewTicker(10 * time.Second),
		pingTimeoutTicker: time.NewTicker(30 * time.Second),
		pongTimeoutTicker: time.NewTicker(30 * time.Second),
	}

	if isServer {
		bootstrap.pingTicker.Stop()
		bootstrap.pongTimeoutTicker.Stop()
	} else {
		bootstrap.pingTimeoutTicker.Stop()
		err := bootstrap.WriteSignal(signal.BootstrapReqSignal{
			AgentId:     config.AppConfig.Agent.Id,
			StaticToken: config.AppConfig.Agent.Proxy.Authentication.StaticToken.Token,
		})
		if err != nil {
			cancel <- err
			return nil
		}
	}

	go bootstrap.handleCommand(ctx)

	return &bootstrap
}

func (b *CtrlConn) handleCommand(ctx context.Context) {
	defer func() {
		_ = recover()
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case <-b.pingTicker.C:
			b.handleSendPing()
		case <-b.pingTimeoutTicker.C:
			b.handlePingTimeout()
		case <-b.pongTimeoutTicker.C:
			b.handlePongTimeout()
		default:
			b.handleSignals()
		}
	}
}

func (b CtrlConn) handleSendPing() {
	if err := b.WriteSignal(signal.PingSignal{}); err != nil {
		b.cancelChan <- err
	} else {
		b.pongTimeoutTicker.Reset(10 * time.Second)
	}
}

func (b CtrlConn) handlePingTimeout() {
	b.cancelChan <- errors.New("heartbeat failure, ping did not receive in time")
}

func (b CtrlConn) handlePongTimeout() {
	b.cancelChan <- errors.New("heartbeat failure, pong did not receive in time")
}

func (b CtrlConn) handleSignals() {
	receivedSignal, err := b.ReadSignal()
	if err != nil || receivedSignal == nil {
		b.cancelChan <- errors.New("error whiling reading signal")
		return
	}

	switch receivedSignal.GetType() {
	case signal.Ping:
		go b.handlePing(*receivedSignal.(*signal.PingSignal))
	case signal.Pong:
		go b.handlePong(*receivedSignal.(*signal.PongSignal))
	case signal.NewConnReq:
		go b.handleNewConnReq(*receivedSignal.(*signal.NewConnReqSignal))
	case signal.BootstrapResp:
		go b.handleBootstrapResp(*receivedSignal.(*signal.BootstrapRespSignal))
	}
}

func (b *CtrlConn) handlePing(pingSignal signal.PingSignal) {
	if err := b.WriteSignal(signal.PongSignal{}); err != nil {
		b.cancelChan <- err
	} else {
		b.pingTimeoutTicker.Stop()
	}
}

func (b *CtrlConn) handlePong(pongSignal signal.PongSignal) {
	b.pongTimeoutTicker.Stop()
}

func (b *CtrlConn) handleNewConnReq(reqSignal signal.NewConnReqSignal) {
	localIp, localPort := util.ResolveAddress(config.AppConfig.Agent.LocalEndpoint)
	localConnection, err := util.Dial(localIp, localPort)
	if err != nil {
		log.Errorf("error dialing local service, reason: %v", err)
		return
	}
	defer localConnection.Close()

	serverIp, serverPort := util.ResolveAddress(config.AppConfig.Agent.ServerEndpoint)
	proxyConnection, err := util.Dial(serverIp, serverPort)
	if err != nil {
		log.Errorf("error crearing proxy connection, reason: %v", err)
		return
	}
	defer proxyConnection.Close()

	wrappedProxyConnection := WrapData(proxyConnection)
	err = wrappedProxyConnection.WriteSignal(signal.NewConnRespSignal{
		AgentId:     config.AppConfig.Agent.Id,
		StaticToken: config.AppConfig.Agent.Proxy.Authentication.StaticToken.Token,
	})
	if err != nil {
		log.Errorf("error writing connection, reason: %v", err)
		return
	}

	wrappedProxyConnection.Join(localConnection)
}

func (b *CtrlConn) handleBootstrapResp(respSignal signal.BootstrapRespSignal) {
	if respSignal.Error != "" {
		b.cancelChan <- errors.New(fmt.Sprintf("error creating bootstrap connection, reason, %v", respSignal.Error))
	}
}
