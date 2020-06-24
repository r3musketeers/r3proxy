package order

import (
	"log"
	"r3-proxy/proxy"
	"time"

	"github.com/google/uuid"
)

type DelayOrderer struct {
	delay   time.Duration
	ordered chan proxy.R3Message
}

func NewDelayOrderer(delay time.Duration) *DelayOrderer {
	return &DelayOrderer{
		delay:   delay,
		ordered: make(chan proxy.R3Message),
	}
}

func (o *DelayOrderer) Propose(message proxy.R3Message) error {
	go func() {
		uuid := uuid.New()
		log.Println("ordering message ", uuid.String())
		time.Sleep(o.delay)
		o.ordered <- message
	}()
	return nil
}

func (o *DelayOrderer) Ordered() <-chan proxy.R3Message {
	return o.ordered
}
