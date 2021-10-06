package connection

import (
	log "github.com/sirupsen/logrus"
	"net"
	"time"
	"tunnel-transporter/internal/signal"
	"tunnel-transporter/internal/util"
)

type rawConn struct {
	net.Conn
}

func wrapRaw(conn net.Conn) *rawConn {
	if _, ok := conn.(*net.TCPConn); ok {
		_ = conn.(*net.TCPConn).SetKeepAlive(true)
		_ = conn.(*net.TCPConn).SetKeepAlivePeriod(30 * time.Second)
	}

	return &rawConn{
		Conn: conn,
	}
}

func (r *rawConn) WriteSignal(typedSignal signal.TypedSignal) error {
	defer func() {
		if err := recover(); err != nil {
			log.Warnf("error writing raw connection, reason: %v", err)
		}
	}()

	if err := util.Write(r, typedSignal); err != nil {
		log.Errorf("error writing raw connection, reason: %v", err)
		return err
	}

	return nil
}

func (r *rawConn) ReadSignal() (signal.TypedSignal, error) {
	defer func() {
		if err := recover(); err != nil {
			log.Warnf("error reading raw connection, reason: %v", err)
		}
	}()

	receivedMessage, err := util.Read(r)
	if err != nil {
		log.Errorf("error reading raw connection, reason: %v", err)
		return nil, err
	}

	return receivedMessage, nil
}

func (r *rawConn) Close() error {
	if r.Conn == nil {
		return nil
	}

	return r.Conn.Close()
}
