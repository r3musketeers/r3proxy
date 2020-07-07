package transport

import (
	"bytes"
	"io"
	"io/ioutil"
	"log"
	"net"

	r3model "r3proxy/model"
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

func (t *TCPTransport) Listen(handle r3model.RequestHandlerFunc) error {
	for {
		conn, err := t.listener.Accept()
		if err != nil {
			return err
		}
		go func() {
			req, err := ioutil.ReadAll(conn)
			if err != nil {
				log.Println(err.Error())
				return
			}
			log.Println(string(req))
			respReader, err := handle(req)
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

func (t *TCPTransport) Deliver(reqReader io.Reader) (io.Reader, error) {
	conn, err := net.Dial("tcp", t.deliverAddr)
	if err != nil {
		return nil, err
	}
	log.Println("connected to service")
	_, err = io.Copy(conn, reqReader)
	if err != nil {
		return nil, err
	}
	log.Println("delivered message")
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
