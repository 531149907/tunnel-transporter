package proxy

import (
	"context"
	"net"
	"tunnel-transporter/util"
)

type DataConnection struct {
	raw *RawConnection
}

func NewDataConnection(ctx context.Context, cancel chan<- error, conn net.Conn) *DataConnection {
	return &DataConnection{raw: NewRawConnection(ctx, cancel, conn)}
}

func (d *DataConnection) join(publicConnection net.Conn) {
	util.Join(d.raw.Conn, publicConnection)
}
