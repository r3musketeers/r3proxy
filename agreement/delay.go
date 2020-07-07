package agreement

import (
	"log"
	"time"

	r3model "r3proxy/model"
)

type DelayConsensusAdapter struct {
	delay   time.Duration
	ordered chan r3model.R3Message
}

func NewDelayConsensusAdapter(delay time.Duration) *DelayConsensusAdapter {
	return &DelayConsensusAdapter{
		delay:   delay,
		ordered: make(chan r3model.R3Message),
	}
}

func (o *DelayConsensusAdapter) Process(message r3model.R3Message) error {
	go func() {
		log.Println("ordering message ", message.ID)
		time.Sleep(o.delay)
		o.ordered <- message
	}()
	return nil
}

func (o *DelayConsensusAdapter) Ordered() <-chan r3model.R3Message {
	return o.ordered
}
