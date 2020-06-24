package main

import (
	"log"
	"time"

	"r3-proxy/order"
	"r3-proxy/transport"
)

func main() {
	tcpTransport, err := transport.NewTCPTransport(":10000", ":10001")
	if err != nil {
		log.Fatal(err.Error())
	}
	delayOrderer := order.NewDelayOrderer(time.Second * 2)
	r3Proxy := NewR3Proxy(tcpTransport, delayOrderer)
	err = r3Proxy.Run()
	if err != nil {
		log.Fatal(err.Error())
	}
}
