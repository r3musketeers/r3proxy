package proxy

import (
	"crypto/sha256"
	"encoding/json"
	"log"
	"sync"
)

type R3Proxy struct {
	transport Transporter
	agreement AgreementAdapter
	clients   sync.Map
	history   sync.Map
}

func NewR3Proxy(transport Transporter, agreement AgreementAdapter) *R3Proxy {
	return &R3Proxy{
		transport: transport,
		agreement: agreement,
	}
}

func (p *R3Proxy) Run(joinAddr string) error {
	if joinAddr != "" {
		joinBody, err := json.Marshal(map[string]string{
			"id":   p.agreement.NodeID(),
			"addr": p.agreement.Address(),
		})
		if err != nil {
			return err
		}
		err = p.transport.SendJoinRequest(joinAddr, joinBody)
		if err != nil {
			return err
		}
	}

	errCh := make(chan error)

	p.agreement.SetHistoryGenerator(p.generateHistory)
	p.agreement.SetHistoryPopulator(p.populateHistory)

	go func() {
		errCh <- p.transport.ListenClient(p.handleClient)
	}()

	go func() {
		errCh <- p.transport.ListenJoin(p.agreement.Join)
	}()

	go func() {
		for message := range p.agreement.Ordered() {
			resp, err := p.transport.Deliver(message.Body)
			if err != nil {
				log.Println(err.Error())
				return
			}
			p.history.Store(message.ID, resp)
			iRespCh, ok := p.clients.Load(message.ID)
			if !ok {
				continue
			}
			respCh := iRespCh.(chan []byte)
			respCh <- resp
		}
	}()

	return <-errCh
}

// Unexported methods

func (p *R3Proxy) handleClient(req []byte) ([]byte, error) {
	hash := sha256.New()
	_, err := hash.Write(req)
	if err != nil {
		return nil, err
	}
	hashSum := string(hash.Sum(nil))

	iResp, ok := p.history.Load(hashSum)
	if ok {
		return iResp.([]byte), nil
	}

	respCh := make(chan []byte)
	errCh := make(chan error)
	p.clients.Store(hashSum, respCh)
	defer func() {
		p.clients.Delete(hashSum)
		close(errCh)
		close(respCh)
	}()
	go func() {
		err := p.agreement.Process(R3Message{
			ID:   hashSum,
			Body: req,
		})
		if err != nil {
			errCh <- err
		}
	}()
	select {
	case resp := <-respCh:
		return resp, nil
	case err := <-errCh:
		return nil, err
	}
}

func (p *R3Proxy) generateHistory() map[string][]byte {
	history := map[string][]byte{}
	p.history.Range(func(iKey, iValue interface{}) bool {
		key := iKey.(string)
		history[key] = iValue.([]byte)
		return true
	})
	return history
}

func (p *R3Proxy) populateHistory(history map[string][]byte) {
	for key, value := range history {
		p.history.Store(key, value)
	}
}
