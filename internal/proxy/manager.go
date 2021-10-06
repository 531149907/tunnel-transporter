package proxy

import (
	"fmt"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"io"
	"net"
	"net/http"
	"time"
	"tunnel-transporter/internal/config"
	"tunnel-transporter/internal/signal"
	"tunnel-transporter/internal/util"
)

type Manager struct {
	proxies        map[string]*Proxy
	unregisterChan chan string
}

func NewManager() *Manager {
	return &Manager{
		proxies:        make(map[string]*Proxy),
		unregisterChan: make(chan string, 10),
	}
}

func (m *Manager) HandleProxyConnection(conn net.Conn) {
	firstMessage, err := util.Read(conn)
	if err != nil || firstMessage == nil {
		log.Errorf("error reading message from connection, reason: %v", err)
		return
	}

	switch firstMessage.GetType() {
	case signal.BootstrapReq:
		m.handleNewControlConnection(*firstMessage.(*signal.BootstrapReqSignal), conn)
	case signal.NewConnResp:
		m.handleNewDataConnection(*firstMessage.(*signal.NewConnRespSignal), conn)
	}
}

func (m *Manager) handleNewControlConnection(reqSignal signal.BootstrapReqSignal, conn net.Conn) {
	if config.AppConfig.Server.Proxy.Authentication.Type == config.StaticToken {
		if reqSignal.StaticToken != config.AppConfig.Server.Proxy.Authentication.StaticToken.Token {
			err := errors.New(fmt.Sprintf("agent %s bootstrap with invalid token %s", reqSignal.AgentId, reqSignal.StaticToken))
			err = util.Write(conn, signal.BootstrapRespSignal{Error: err.Error()})
			if err != nil {
				conn.Close()
				return
			}

			log.Errorf("%v", err)
			select {
			case <-time.After(10 * time.Second):
				conn.Close()
			}
		}
	}

	proxy, err := NewProxy(reqSignal.AgentId, reqSignal.AgentVersion, conn, m.unregisterChan)
	if err != nil {
		log.Errorf("error handling bootstrap connection, reason: %v", err)
		return
	}

	m.proxies[proxy.agentId] = proxy
}

func (m *Manager) handleNewDataConnection(respSignal signal.NewConnRespSignal, conn net.Conn) {
	if proxy, ok := m.proxies[respSignal.AgentId]; !ok {
		log.Warnf("fail to find tunnel proxy for agent %s", respSignal.AgentId)
	} else {
		proxy.handleNewDataConnection(respSignal, conn)
	}
}

func (m *Manager) HandlePublicHTTPConnection(rawRequest *http.Request, rawWriter http.ResponseWriter) {
	agentId := rawRequest.Header.Get(config.AppConfig.Server.Http.AgentIdHeaderKey)
	if agentId == "" {
		log.Info("agent id header key not found in request")
		rawWriter.WriteHeader(http.StatusNotFound)
		return
	}

	if _, ok := m.proxies[agentId]; !ok {
		log.Warnf("agent id %s not found in manager", agentId)
		rawWriter.WriteHeader(http.StatusNotFound)
		return
	}

	req, err := http.NewRequest(rawRequest.Method, "http://"+rawRequest.Host+rawRequest.RequestURI, rawRequest.Body)
	if err != nil {
		log.Errorf("error handling new http request, reason: %v", err)
		rawWriter.WriteHeader(http.StatusInternalServerError)
		return
	}

	for k, v := range rawRequest.Header {
		req.Header[k] = v
	}

	resp, err := m.proxies[agentId].httpClient.Do(req)
	if err != nil {
		log.Errorf("error handling new http request, reason: %v", err)
		rawWriter.WriteHeader(http.StatusInternalServerError)
		return
	}

	defer resp.Body.Close()

	_, err = io.Copy(rawWriter, resp.Body)
	if err != nil {
		log.Errorf("error copying connection, reason: %v", err)
		return
	}
}

func (m *Manager) handleUnregister() {
	for {
		select {
		case agentId, ok := <-m.unregisterChan:
			if !ok {
				return
			}

			delete(m.proxies, agentId)
		}
	}
}
