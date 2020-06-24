package proxy

import (
	"io"
	"log"
	"sync"

	"github.com/google/uuid"
)

type Transporter interface {
	Listen(HandlerFunc) error
	Deliver(io.Reader) (io.Reader, error)
}

type Orderer interface {
	Propose(R3Message) error
	Ordered() <-chan R3Message
}

type R3Proxy struct {
	transport Transporter
	orderer   Orderer
	clients   sync.Map
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
			log.Println("message ordered ", message.ID)
			respReader, err := p.transport.Deliver(message.Reader)
			if err != nil || respReader == nil {
				errCh <- err
			}
			respChI, ok := p.clients.Load(message.ID)
			if !ok {
				log.Println("response for non existing client")
				continue
			}
			respCh := respChI.(chan io.Reader)
			respCh <- respReader
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
	p.clients.Store(uuidStr, respCh)
	log.Println("stored client ", uuidStr)
	return <-respCh, nil
}
