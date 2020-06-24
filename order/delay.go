package order

import (
	"log"
	"r3-proxy/proxy"
	"time"
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
		log.Println("ordering message ", message.ID)
		time.Sleep(o.delay)
		o.ordered <- message
	}()
	return nil
}

func (o *DelayOrderer) Ordered() <-chan proxy.R3Message {
	return o.ordered
}
