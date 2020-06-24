package main

import (
	"io"
	"log"
	"net"
)

type Transporter interface {
	Listen(func(net.Conn) error) error
	Deliver(io.Reader) (io.Reader, error)
}

type Orderer interface {
	Propose(net.Conn) error
	Ordered() <-chan io.ReadWriteCloser
}

type R3Proxy struct {
	transport Transporter
	orderer   Orderer
}

func NewR3Proxy(transport Transporter, orderer Orderer) *R3Proxy {
	return &R3Proxy{
		transport: transport,
		orderer:   orderer,
	}
}

func (p *R3Proxy) Run() error {
	errCh := make(chan error)
	go func() {
		errCh <- p.transport.Listen(p.orderer.Propose)
	}()
	go func() {
		for inStream := range p.orderer.Ordered() {
			outStream, err := p.transport.Deliver(inStream)
			if err != nil {
				errCh <- err
			}
			_, err = io.Copy(inStream, outStream)
			if err != nil {
				log.Println(err.Error())
			}
			inStream.Close()
		}
	}()
	return <-errCh
}
