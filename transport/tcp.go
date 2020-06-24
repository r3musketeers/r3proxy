package transport

import (
	"bytes"
	"io"
	"io/ioutil"
	"log"
	"net"
)

type TCPTransport struct {
	listener    net.Listener
	deliverAddr string
}

func NewTCPTransport(listenAddr, deliverAddr string) (*TCPTransport, error) {
	ln, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return nil, err
	}
	return &TCPTransport{
		listener:    ln,
		deliverAddr: deliverAddr,
	}, nil
}

func (t *TCPTransport) Listen(handle func(io.Reader) (io.Reader, error)) error {
	for {
		conn, err := t.listener.Accept()
		if err != nil {
			return err
		}
		go func() {
			reqPayload, err := ioutil.ReadAll(conn)
			if err != nil {
				log.Println(err.Error())
				return
			}
			respReader, err := handle(bytes.NewBuffer(reqPayload))
			if err != nil {
				log.Println(err.Error())
				conn.Close()
				return
			}
			_, err = io.Copy(conn, respReader)
			if err != nil {
				log.Println(err.Error())
			}
			conn.Close()
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
