package proxy

import (
	"context"
	log "github.com/sirupsen/logrus"
	"net"
	"tunnel-transporter/config"
	"tunnel-transporter/constants"
	"tunnel-transporter/message"
	"tunnel-transporter/util"
)

type Proxy struct {
	AgentId      string
	AgentVersion string

	PublicListener   *net.TCPListener
	PublicListenPort uint16

	BootstrapConnection *BootstrapConnection
	Connections         []*DataConnection
	ConnectionsChan     chan *DataConnection

	rootContext context.Context
	rootCancel  context.CancelFunc

	cancel  chan error
	closing bool
}

func NewProxy(agentId string, agentVersion string, conn *net.TCPConn, unregisterChan chan<- uint16) (*Proxy, error) {
	cancelChan := make(chan error)
	ctx, cancel := context.WithCancel(context.Background())

	listener, port, err := newListener()
	if err != nil {
		log.Errorf("error creating listener, reason: %v", err)
		cancel()
		close(cancelChan)
		return nil, err
	}

	log.Infof("starting tunnel for agent %s, using port %d", agentId, port)

	tunnelProxy := Proxy{
		AgentId:             agentId,
		AgentVersion:        agentVersion,
		PublicListener:      listener,
		PublicListenPort:    uint16(port),
		BootstrapConnection: NewBootstrapConnection(ctx, cancelChan, conn, true),
		Connections:         []*DataConnection{},
		ConnectionsChan:     make(chan *DataConnection, 10),
		rootContext:         ctx,
		rootCancel:          cancel,
		cancel:              cancelChan,
	}

	go tunnelProxy.handlePublicConnection(ctx)
	go tunnelProxy.shutdown(unregisterChan)

	return &tunnelProxy, nil
}

func newListener() (listener *net.TCPListener, port int, err error) {
	listener, err = util.ListenOnRandomPort()
	if err != nil {
		log.Errorf("error while listening on %s, reason: %v", listener.Addr().String(), err)
		return nil, 0, err
	}

	_, port = util.ResolveAddress(listener.Addr().String())
	return
}

func (t *Proxy) handlePublicConnection(ctx context.Context) {
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("error handing public connection, reason: %v", err)
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		default:
			publicConnection, err := t.PublicListener.AcceptTCP()
			if err != nil {
				log.Errorf("error while accepting public connection, reason: %v", err)
				continue
			}

			go func() {
				select {
				case <-ctx.Done():
					return
				default:
					t.BootstrapConnection.raw.write(message.RequireNewConnectionRequestMessage{})

					proxyConnection, ok := <-t.ConnectionsChan
					if !ok {
						log.Errorf("error reading proxy connection channel, agentId %s", t.AgentId)
						return
					}

					log.Debug("public connection connected to proxy connection")
					proxyConnection.join(publicConnection)
				}
			}()
		}
	}
}

func (t *Proxy) HandleNewDataConnection(responseMessage message.RequireNewConnectionResponseMessage, conn *net.TCPConn) {
	if config.ClientConfig.Server.Authentication.Type == constants.StaticToken {
		if responseMessage.StaticToken != config.ClientConfig.Server.Authentication.StaticToken.Token {
			conn.Close()
			return
		}
	}

	ctx, _ := context.WithCancel(t.rootContext)

	newDataConnection := NewDataConnection(ctx, t.cancel, conn)
	t.ConnectionsChan <- newDataConnection
	t.Connections = append(t.Connections, newDataConnection)
}

func (t *Proxy) shutdown(unregisterChan chan<- uint16) {
	select {
	case err := <-t.cancel:
		if t.closing {
			return
		}

		log.Infof("===> shutting down tunnel (agent %s) due to error: %v", t.AgentId, err)

		t.closing = true
		t.rootCancel()
		unregisterChan <- t.PublicListenPort
		close(t.cancel)
		t.PublicListener.Close()
		close(t.ConnectionsChan)

		log.Infof("===> completed shutting down tunnel (agent %s)", t.AgentId)
	}
}
