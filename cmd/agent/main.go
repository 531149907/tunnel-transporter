package agent

import (
	"context"
	log "github.com/sirupsen/logrus"
	"time"
	"tunnel-transporter/internal/config"
	"tunnel-transporter/internal/connection"
	"tunnel-transporter/internal/util"
)

var (
	ctx        context.Context
	cancel     context.CancelFunc
	cancelChan chan error
	closing    bool
)

func RunAgent() {
	for {
		ctx, cancel = context.WithCancel(context.Background())
		cancelChan = make(chan error)
		closing = false

		serverIp, serverPort := util.ResolveAddress(config.AppConfig.Agent.ServerEndpoint)
		conn, err := util.Dial(serverIp, serverPort)
		if err != nil {
			log.Errorf("error dialing tcp, reason: %v", err)
			cancelChan <- err
		} else {
			connection.NewCtrl(ctx, cancelChan, conn, false)
		}

		shutdown()

		time.Sleep(5 * time.Second)
	}
}

func shutdown() {
	select {
	case err := <-cancelChan:
		if closing {
			return
		}

		log.Errorf("shutting down agent due to error: %v", err)

		closing = true
		close(cancelChan)
		cancel()

		log.Errorf("completed shutting down agent")
	}
}
