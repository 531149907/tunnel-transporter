package util

import (
	"encoding/binary"
	"io"
	"net"
	"strconv"
	"sync"
	"tunnel-transporter/message"
)

func Dial(host string, port int) (*net.TCPConn, error) {
	addr, _ := net.ResolveTCPAddr("tcp", host+":"+strconv.Itoa(port))
	conn, err := net.DialTCP("tcp", nil, addr)
	if err != nil {
		return nil, err
	}

	return conn, err
}

func Listen(port int) (*net.TCPListener, error) {
	addr, _ := net.ResolveTCPAddr("tcp", ":"+strconv.Itoa(port))
	return net.ListenTCP("tcp", addr)
}

func ListenOnRandomPort() (*net.TCPListener, error) {
	return net.ListenTCP("tcp", nil)
}

func ResolveAddress(address string) (string, int) {
	addr, err := net.ResolveTCPAddr("tcp", address)
	if err != nil {
		return "", -1
	}

	return addr.IP.String(), addr.Port
}

func Join(to net.Conn, from net.Conn) {
	var wait sync.WaitGroup

	pipe := func(to net.Conn, from net.Conn) {
		defer to.Close()
		defer from.Close()
		defer wait.Done()

		if _, err := io.Copy(to, from); err != nil {
			return
		}
	}

	wait.Add(2)

	go pipe(from, to)
	go pipe(to, from)

	wait.Wait()
}

func Read(conn net.Conn) (message.TypedMessage, error) {
	var size int64
	if err := binary.Read(conn, binary.LittleEndian, &size); err != nil {
		return nil, err
	}

	buffer := make([]byte, size)
	if n, err := conn.Read(buffer); err != nil || int64(n) != size {
		return nil, err
	}

	msg, err := message.Unpack(buffer)
	if err != nil {
		return nil, err
	}

	return msg, nil
}

func Write(conn net.Conn, typedMessage message.TypedMessage) error {
	buffer, err := message.Pack(typedMessage)
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
