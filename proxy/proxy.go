package proxy

import (
	"bytes"
	"io"
	"log"
	"sync"

	r3model "r3proxy/model"

	"github.com/google/uuid"
)

type Transporter interface {
	ListenRequest(r3model.RequestHandlerFunc) error
	ListenJoin(r3model.JoinHandlerFunc) error
	Deliver(io.Reader) (io.Reader, error)
}

type AgreementAdapter interface {
	Join(string, string) error
	Process(r3model.R3Message) error
	Ordered() <-chan r3model.R3Message
}

type R3Proxy struct {
	transport Transporter
	agreement AgreementAdapter
	clients   sync.Map
}

func NewR3Proxy(transport Transporter, agreement AgreementAdapter) *R3Proxy {
	return &R3Proxy{
		transport: transport,
		agreement: agreement,
	}
}

func (p *R3Proxy) Run() error {
	errCh := make(chan error)
	go func() {
		errCh <- p.transport.ListenRequest(p.handleRequest)
	}()
	go func() {
		errCh <- p.transport.ListenJoin(p.agreement.Join)
	}()
	go func() {
		for message := range p.agreement.Ordered() {
			log.Println("message ordered", message.ID)
			log.Println("message body", message.Body)
			respReader, err := p.transport.Deliver(bytes.NewBuffer(message.Body))
			if err != nil || respReader == nil {
				errCh <- err
				return
			}
			respChI, ok := p.clients.Load(message.ID)
			if !ok {
				log.Println("response for non existing client")
				continue
			}
			log.Println("returning response")
			respCh := respChI.(chan io.Reader)
			respCh <- respReader
		}
	}()
	return <-errCh
}

func (p *R3Proxy) handleRequest(req []byte) (io.Reader, error) {
	uuidStr := uuid.New().String()
	respCh := make(chan io.Reader)
	errCh := make(chan error)
	p.clients.Store(uuidStr, respCh)
	defer func() {
		p.clients.Delete(uuidStr)
		close(errCh)
		close(respCh)
	}()
	go func() {
		err := p.agreement.Process(r3model.R3Message{
			ID:   uuidStr,
			Body: req,
		})
		if err != nil {
			errCh <- err
		}
	}()
	select {
	case respReader := <-respCh:
		return respReader, nil
	case err := <-errCh:
		return nil, err
	}
}
