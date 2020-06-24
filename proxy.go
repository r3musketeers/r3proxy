package main

import (
	"io"
	"sync"

	"github.com/google/uuid"
)

type Transporter interface {
	Listen(func(io.Reader) (io.Reader, error)) error
	Deliver(io.Reader) (io.Reader, error)
}

type Orderer interface {
	Propose(string, io.Reader) error
	Ordered() <-chan io.Reader
}

type R3Proxy struct {
	transport      Transporter
	orderer        Orderer
	pendingClients sync.Map
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
		errCh <- p.transport.Listen(p.handle)
	}()
	go func() {
		for inStream := range p.orderer.Ordered() {
			outStream, err := p.transport.Deliver(inStream)
			if err != nil {
				errCh <- err
			}
			p.pendingClients.Load()
		}
	}()
	return <-errCh
}

func (p *R3Proxy) handle(reqReader io.Reader) (io.Reader, error) {
	uuidStr := uuid.New().String()
	err := p.orderer.Propose(uuidStr, reqReader)
	if err != nil {
		return nil, err
	}
	respCh := make(chan io.Reader)
	p.pendingClients.Store(uuidStr, respCh)
	return <-respCh, nil
}
