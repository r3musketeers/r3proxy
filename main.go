package main

import (
	"log"
	"time"

	r3raft "r3proxy/agreement/raft"
	r3proxy "r3proxy/proxy"
	"r3proxy/transport"

	"github.com/spf13/pflag"
)

var (
	listenClientAddr = pflag.String(
		"listen-client", ":10010", "Address to listen client requests",
	)
	listenJoinAddr = pflag.String(
		"listen-join", ":10011", "Address to listen join requests",
	)
	deliverAddr = pflag.String(
		"deliver-addr", ":20010", "Address to deliver ordered requests",
	)
	agreementAddr = pflag.String(
		"agreement-addr", ":10012", "Agreement address binding",
	)
	joinAddr = pflag.String(
		"join-addr", "", "Address to join an existing cluster",
	)
)

func main() {
	pflag.Parse()
	if pflag.NArg() == 0 {
		log.Fatal("node id not specified")
	}
	nodeID := pflag.Arg(0)

	httpTransport := transport.NewHTTPTransport(*listenClientAddr, *listenJoinAddr, *deliverAddr)

	raftAgreement, err := r3raft.NewRaftAgreementAdapter(
		nodeID,
		*agreementAddr,
		10*time.Second,
		"data/raft/"+nodeID,
		2,
		10*time.Second,
		*joinAddr == "",
	)

	r3Proxy := r3proxy.NewR3Proxy(httpTransport, raftAgreement)
	err = r3Proxy.Run(*joinAddr)
	if err != nil {
		log.Fatal(err.Error())
	}
}
