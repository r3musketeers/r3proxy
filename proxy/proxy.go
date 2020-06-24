package proxy

import (
	"io"
	"log"
	"sync"

	"github.com/google/uuid"
)

type Transporter interface {
	Listen(func(io.Reader) (io.Reader, error)) error
	Deliver(io.Reader) (io.Reader, error)
}

type Orderer interface {
	Propose(R3Message) error
	Ordered() <-chan R3Message
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
		for message := range p.orderer.Ordered() {
			outStream, err := p.transport.Deliver(message.Reader)
			if err != nil {
				errCh <- err
			}
			respChI, ok := p.pendingClients.Load(message.ID)
			if !ok {
				log.Println("response for non existing client")
				continue
			}
			respCh := respChI.(chan io.Reader)
			respCh <- outStream
		}
	}()
	return <-errCh
}

func (p *R3Proxy) handle(reqReader io.Reader) (io.Reader, error) {
	uuidStr := uuid.New().String()
	err := p.orderer.Propose(R3Message{
		ID:     uuidStr,
		Reader: reqReader,
	})
	if err != nil {
		return nil, err
	}
	respCh := make(chan io.Reader)
	p.pendingClients.Store(uuidStr, respCh)
	return <-respCh, nil
}
