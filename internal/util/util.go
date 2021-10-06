package util

import (
	"encoding/binary"
	"net"
	"strconv"
	"tunnel-transporter/internal/signal"
)

func Dial(host string, port int) (net.Conn, error) {
	addr, _ := net.ResolveTCPAddr("tcp", host+":"+strconv.Itoa(port))
	conn, err := net.DialTCP("tcp", nil, addr)
	if err != nil {
		return nil, err
	}

	return conn, err
}

func Listen(port int) (net.Listener, error) {
	addr, _ := net.ResolveTCPAddr("tcp", ":"+strconv.Itoa(port))
	return net.Listen("tcp", addr.String())
}

func ListenOnRandomPort() (net.Listener, error) {
	return net.Listen("tcp", "")
}

func ResolveAddress(address string) (string, int) {
	addr, err := net.ResolveTCPAddr("tcp", address)
	if err != nil {
		return "", -1
	}

	return addr.IP.String(), addr.Port
}

func Read(conn net.Conn) (signal.TypedSignal, error) {
	var size int64
	if err := binary.Read(conn, binary.LittleEndian, &size); err != nil {
		return nil, err
	}

	buffer := make([]byte, size)
	if n, err := conn.Read(buffer); err != nil || int64(n) != size {
		return nil, err
	}

	msg, err := signal.Unpack(buffer)
	if err != nil {
		return nil, err
	}

	return msg, nil
}

func Write(conn net.Conn, typedMessage signal.TypedSignal) error {
	buffer, err := signal.Pack(typedMessage)
	if err != nil {
		return err
	}

	if err = binary.Write(conn, binary.LittleEndian, int64(len(buffer))); err != nil {
		return err
	}

	if _, err = conn.Write(buffer); err != nil {
		return err
	}

	return nil
}
