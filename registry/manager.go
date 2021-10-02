package registry

import (
	"sync"
	"tunnel-transporter/proxy"
)

type Manager struct {
	proxies        sync.Map
	agentIdPortMap sync.Map
	UnregisterChan chan uint16
}

func NewRegistryManager() *Manager {
	manager := &Manager{
		UnregisterChan: make(chan uint16, 10),
	}

	return manager
}

func (m *Manager) Put(publicListenPort uint16, tunnelProxy *proxy.Proxy) {
	m.proxies.Store(publicListenPort, tunnelProxy)
	m.agentIdPortMap.Store(tunnelProxy.AgentId, publicListenPort)
}

func (m *Manager) Remove(publicListenPort uint16) {
	v, ok := m.proxies.LoadAndDelete(publicListenPort)
	if !ok {
		return
	}

	m.agentIdPortMap.Delete(v.(*proxy.Proxy).AgentId)
}

func (m *Manager) Contains(publicListenPort uint16) bool {
	_, ok := m.proxies.Load(publicListenPort)
	return ok
}

func (m *Manager) Get(publicListenPort uint16) *proxy.Proxy {
	v, ok := m.proxies.Load(publicListenPort)
	if ok {
		return v.(*proxy.Proxy)
	} else {
		return nil
	}
}

func (m *Manager) GetByAgentId(agentId string) *proxy.Proxy {
	v, ok := m.agentIdPortMap.Load(agentId)
	if !ok {
		return nil
	}
	return m.Get(v.(uint16))
}

func (m *Manager) unregister() {
	for {
		select {
		case port := <-m.UnregisterChan:
			m.Remove(port)
		}
	}
}
