package transport

import (
	"bytes"
	"io"
	"log"
	"net"
)

type TCPTransport struct {
	listener    net.Listener
	deliverAddr string
}

func NewTCPTransport(listenAddr string, deliverAddr string) (*TCPTransport, error) {
	ln, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return nil, err
	}
	return &TCPTransport{
		listener:    ln,
		deliverAddr: deliverAddr,
	}, nil
}

func (t *TCPTransport) Listen(handleConn func(net.Conn) error) error {
	for {
		conn, err := t.listener.Accept()
		if err != nil {
			return err
		}
		go func() {
			err = handleConn(conn)
			if err != nil {
				log.Println(err.Error())
			}
		}()
	}
}

func (t *TCPTransport) Deliver(inStream io.Reader) (io.Reader, error) {
	conn, err := net.Dial("tcp", t.deliverAddr)
	if err != nil {
		return nil, err
	}
	_, err = io.Copy(conn, inStream)
	if err != nil {
		return nil, err
	}
	outStream := bytes.NewBuffer([]byte{})
	_, err = io.Copy(outStream, conn)
	if err != nil {
		return nil, err
	}
	err = conn.Close()
	if err != nil {
		return nil, err
	}
	return outStream, nil
}
