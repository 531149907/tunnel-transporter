package proxy

import (
	"context"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"net"
	"net/http"
	"time"
	"tunnel-transporter/internal/config"
	"tunnel-transporter/internal/connection"
	"tunnel-transporter/internal/signal"
	"tunnel-transporter/internal/util"
)

type Proxy struct {
	agentId      string
	agentVersion string

	publicListener   net.Listener
	publicListenPort uint16

	ctrlConn     *connection.CtrlConn
	dataConns    []*connection.DataConn
	dataConnChan chan *connection.DataConn

	httpClient http.Client

	rootContext context.Context
	rootCancel  context.CancelFunc

	cancel  chan error
	closing bool
}

func NewProxy(agentId string, agentVersion string, conn net.Conn, unregisterChan chan<- string) (*Proxy, error) {
	listener, port, err := newListener()
	if err != nil {
		log.Errorf("error creating listener, reason: %v", err)
		return nil, err
	}

	log.Infof("<== starting tunnel for agent %s, using port %d", agentId, port)

	cancelChan := make(chan error)
	ctx, cancel := context.WithCancel(context.Background())

	tunnelProxy := Proxy{
		agentId:          agentId,
		agentVersion:     agentVersion,
		publicListener:   listener,
		publicListenPort: uint16(port),
		ctrlConn:         connection.NewCtrl(ctx, cancelChan, conn, true),
		dataConns:        []*connection.DataConn{},
		dataConnChan:     make(chan *connection.DataConn, 10),
		rootContext:      ctx,
		rootCancel:       cancel,
		cancel:           cancelChan,
		closing:          false,
	}

	tunnelProxy.httpClient = http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				proxyConnection, err := tunnelProxy.getProxyConnection()
				if err != nil {
					log.Errorf("error reading proxy connection channel in agentId %s, reason: %v", tunnelProxy.agentId, err)
					return nil, err
				}

				return proxyConnection, nil
			},
		},
	}

	go tunnelProxy.shutdown(unregisterChan)
	go tunnelProxy.handlePublicConnection(ctx)

	return &tunnelProxy, nil
}

func newListener() (listener net.Listener, port int, err error) {
	listener, err = util.ListenOnRandomPort()
	if err != nil {
		log.Errorf("error while listening on %s, reason: %v", listener.Addr().String(), err)
		return nil, 0, err
	}

	_, port = util.ResolveAddress(listener.Addr().String())
	return
}

func (p *Proxy) handlePublicConnection(ctx context.Context) {
	defer func() {
		_ = recover()
	}()

	for {
		select {
		case <-ctx.Done():
			return
		default:
			publicConnection, err := p.publicListener.Accept()
			if err != nil {
				log.Errorf("error while accepting public connection, reason: %v", err)
				continue
			}

			go func() {
				proxyConnection, err := p.getProxyConnection()
				if err != nil {
					log.Errorf("error obtaining proxy connection in agent %s, reason: %v", p.agentId, err)
					return
				}

				proxyConnection.Join(publicConnection)
			}()
		}
	}
}

func (p *Proxy) getProxyConnection() (*connection.DataConn, error) {
	if err := p.ctrlConn.WriteSignal(signal.NewConnReqSignal{}); err != nil {
		return nil, err
	}

	timeoutTicker := time.NewTicker(10 * time.Second)
	select {
	case <-timeoutTicker.C:
		return nil, errors.New("fetching proxy connection timeout")
	case proxyConnection, ok := <-p.dataConnChan:
		if !ok {
			return nil, errors.New("data connection channel closed")
		}

		timeoutTicker.Stop()
		return proxyConnection, nil
	}
}

func (p *Proxy) handleNewDataConnection(responseMessage signal.NewConnRespSignal, conn net.Conn) {
	if config.AppConfig.Server.Proxy.Authentication.Type == config.StaticToken {
		if responseMessage.StaticToken != config.AppConfig.Server.Proxy.Authentication.StaticToken.Token {
			conn.Close()
			return
		}
	}

	newDataConnection := connection.WrapData(conn)
	p.dataConnChan <- newDataConnection
	p.dataConns = append(p.dataConns, newDataConnection)
}

func (p *Proxy) shutdown(unregisterChan chan<- string) {
	select {
	case err := <-p.cancel:
		if p.closing {
			return
		}

		log.Infof("===> shutting down tunnel (agent %s) due to error: %v", p.agentId, err)

		p.closing = true
		close(p.cancel)
		p.rootCancel()
		p.publicListener.Close()
		_ = p.ctrlConn.Close()
		close(p.dataConnChan)
		unregisterChan <- p.agentId
		for _, conn := range p.dataConns {
			_ = conn.Close()
		}
	}
}
