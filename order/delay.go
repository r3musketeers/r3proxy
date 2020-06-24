package order

import (
	"io"
	"log"
	"net"
	"time"

	"github.com/google/uuid"
)

type DelayOrderer struct {
	delay   time.Duration
	ordered chan io.ReadWriteCloser
}

func NewDelayOrderer(delay time.Duration) *DelayOrderer {
	return &DelayOrderer{
		delay:   delay,
		ordered: make(chan io.ReadWriteCloser),
	}
}

func (o *DelayOrderer) Propose(conn net.Conn) error {
	go func() {
		uuid := uuid.New()
		log.Println("ordering message ", uuid.String())
		time.Sleep(o.delay)
		o.ordered <- conn
	}()
	return nil
}

func (o *DelayOrderer) Ordered() <-chan io.ReadWriteCloser {
	return o.ordered
}
