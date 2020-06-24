package order

import (
	"io"
	"io/ioutil"
	"log"
	"time"

	"github.com/google/uuid"
)

type DelayOrderer struct {
	delay   time.Duration
	ordered chan io.ReadCloser
}

func NewDelayOrderer(delay time.Duration) *DelayOrderer {
	return &DelayOrderer{
		delay:   delay,
		ordered: make(chan io.ReadCloser),
	}
}

func (o *DelayOrderer) Propose(inStream io.Reader) error {
	go func() {
		uuid := uuid.New()
		log.Println("ordering message ", uuid.String())
		time.Sleep(o.delay)
		o.ordered <- ioutil.NopCloser(inStream)
	}()
	return nil
}

func (o *DelayOrderer) Ordered() <-chan io.ReadWriteCloser {
	return o.ordered
}
