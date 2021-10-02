package client

import (
	"context"
	log "github.com/sirupsen/logrus"
	"time"
	"tunnel-transporter/config"
	"tunnel-transporter/proxy"
	"tunnel-transporter/util"
)

var (
	ctx        context.Context
	cancel     context.CancelFunc
	cancelChan chan error
	closing    bool
)

func StartAgent() {
	for {
		ctx, cancel = context.WithCancel(context.Background())
		cancelChan = make(chan error)
		closing = false

		serverIp, serverPort := util.ResolveAddress(config.ClientConfig.Agent.ServerEndpoint)
		conn, err := util.Dial(serverIp, serverPort)
		if err != nil {
			log.Errorf("error dialing tcp, reason: %v", err)
			cancelChan <- err
		} else {
			proxy.NewBootstrapConnection(ctx, cancelChan, conn, false)
		}

		shutdown()
	}
}

func shutdown() {
	select {
	case err := <-cancelChan:
		if closing {
			return
		}

		closing = true
		log.Errorf("shutting down agent due to error: %v", err)

		close(cancelChan)
		cancel()

		log.Errorf("completed shutting down agent")
	}

	time.Sleep(5 * time.Second)
}
