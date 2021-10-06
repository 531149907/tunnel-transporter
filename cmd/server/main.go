package server

import (
	"context"
	log "github.com/sirupsen/logrus"
	"net/http"
	"strconv"
	"tunnel-transporter/internal/config"
	"tunnel-transporter/internal/proxy"
	"tunnel-transporter/internal/util"
)

func RunServer(ctx context.Context) {
	go startAgentBootstrapServer()
	go startPublicHTTPServer()

	select {
	case <-ctx.Done():
		return
	}
}

var proxyManager = proxy.NewManager()

func startAgentBootstrapServer() {
	listener, err := util.Listen(int(config.AppConfig.Server.AgentPort))
	if err != nil {
		_, port := util.ResolveAddress(listener.Addr().String())
		log.Panicf("error while listening on %d, reason: %v", port, err)
	}

	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Errorf("error while accepting connection, reason: %v", err)
			continue
		}

		go proxyManager.HandleProxyConnection(conn)
	}
}

func startPublicHTTPServer() {
	httpServer := http.NewServeMux()
	httpServer.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		proxyManager.HandlePublicHTTPConnection(request, writer)
	})

	err := http.ListenAndServe(":"+strconv.Itoa(int(config.AppConfig.Server.Http.Port)), httpServer)
	if err != nil {
		log.Panicf("error while listening on %d, reason: %v", config.AppConfig.Server.Http.Port, err)
		return
	}
}
