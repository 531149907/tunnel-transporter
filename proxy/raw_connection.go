package proxy

import (
	"context"
	log "github.com/sirupsen/logrus"
	"net"
	"tunnel-transporter/message"
	"tunnel-transporter/util"
)

type RawConnection struct {
	net.Conn
	cancel chan<- error
}

func NewRawConnection(ctx context.Context, cancel chan<- error, conn net.Conn) *RawConnection {
	rawConnection := &RawConnection{
		Conn:   conn,
		cancel: cancel,
	}

	_, ok := conn.(*net.TCPConn)
	if ok {
		err := conn.(*net.TCPConn).SetKeepAlive(true)
		if err != nil {
			return nil
		}
	}

	go rawConnection.shutdown(ctx)

	return rawConnection
}

func (r *RawConnection) write(typedMessage message.TypedMessage) {
	defer func() {
		if err := recover(); err != nil {
			log.Warnf("error writing raw connection, reason: %v", err)
		}
	}()

	if err := util.Write(r.Conn, typedMessage); err != nil {
		log.Errorf("error writing raw connection, reason: %v", err)
		r.cancel <- err
	}
}

func (r *RawConnection) read() message.TypedMessage {
	defer func() {
		if err := recover(); err != nil {
			log.Warnf("error reading raw connection, reason: %v", err)
		}
	}()

	receivedMessage, err := util.Read(r.Conn)
	if err != nil {
		log.Errorf("error reading raw connection, reason: %v", err)
		r.cancel <- err
		return nil
	}

	return receivedMessage
}

func (r *RawConnection) shutdown(ctx context.Context) {
	select {
	case <-ctx.Done():
		if r.Conn != nil {
			_ = r.Conn.Close()
		}
	}
}
