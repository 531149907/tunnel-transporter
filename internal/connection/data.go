package connection

import (
	"io"
	"net"
	"sync"
)

type DataConn struct {
	*rawConn
}

func WrapData(conn net.Conn) *DataConn {
	return &DataConn{rawConn: wrapRaw(conn)}
}

func (d *DataConn) Join(publicConnection net.Conn) {
	defer d.Close()
	defer publicConnection.Close()

	var wait sync.WaitGroup

	pipe := func(to net.Conn, from net.Conn) {
		defer wait.Done()
		_, _ = io.Copy(to, from)
	}

	wait.Add(2)

	go pipe(d, publicConnection)
	go pipe(publicConnection, d)

	wait.Wait()
}
