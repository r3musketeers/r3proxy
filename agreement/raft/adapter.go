package raft

import (
	"bytes"
	"encoding/gob"
	"errors"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"time"

	r3errors "r3proxy/errors"
	r3model "r3proxy/model"

	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb"
)

type RaftAgreementAdapter struct {
	raft            *raft.Raft
	proposalTimeout time.Duration
	orderedCh       chan r3model.R3Message
}

func NewRaftAgreementAdapter(
	nodeID string,
	addrStr string,
	transportTimeout time.Duration,
	baseDir string,
	snapshotRetain int,
	proposalTimeout time.Duration,
	joinAddrStr string,
) (*RaftAgreementAdapter, error) {
	config := raft.DefaultConfig()
	config.LocalID = raft.ServerID(nodeID)

	addr, err := net.ResolveTCPAddr("tcp", addrStr)
	if err != nil {
		return nil, err
	}
	transport, err := raft.NewTCPTransport(
		addrStr,
		addr,
		3,
		10*time.Second,
		os.Stderr,
	)
	if err != nil {
		return nil, err
	}

	snapshotStore, err := raft.NewFileSnapshotStore(
		baseDir,
		snapshotRetain,
		os.Stderr,
	)
	if err != nil {
		return nil, err
	}

	boltDB, err := raftboltdb.NewBoltStore(filepath.Join(baseDir, "raft.db"))
	if err != nil {
		return nil, err
	}

	raftAdapter := &RaftAgreementAdapter{
		proposalTimeout: proposalTimeout,
		orderedCh:       make(chan r3model.R3Message),
	}

	raftInstance, err := raft.NewRaft(
		config,
		raftAdapter,
		boltDB,
		boltDB,
		snapshotStore,
		transport,
	)
	if err != nil {
		return nil, err
	}

	raftAdapter.raft = raftInstance

	if joinAddrStr == "" {
		configSingle := raft.Configuration{
			Servers: []raft.Server{
				{ID: config.LocalID, Address: transport.LocalAddr()},
			},
		}
		raftInstance.BootstrapCluster(configSingle)
	}

	return raftAdapter, nil
}

////////////////////////////////////////////////////////////////////////////////
//
// r3Proxy.ConsensusAdapter interface implementation
//
////////////////////////////////////////////////////////////////////////////////

func (a *RaftAgreementAdapter) Join(nodeID, addrStr string) error {
	configFuture := a.raft.GetConfiguration()
	err := configFuture.Error()
	if err != nil {
		return err
	}
	for _, server := range configFuture.Configuration().Servers {
		if server.ID == raft.ServerID(nodeID) || server.Address == raft.ServerAddress(addrStr) {
			if server.ID == raft.ServerID(nodeID) && server.Address == raft.ServerAddress(addrStr) {
				log.Printf("node %s at %s already a member, ignoring request", nodeID, addrStr)
				return nil
			}
			removeFuture := a.raft.RemoveServer(server.ID, 0, 0)
			err = removeFuture.Error()
			if err != nil {
				return err
			}
		}
	}
	addVoterFuture := a.raft.AddVoter(raft.ServerID(nodeID), raft.ServerAddress(addrStr), 0, 0)
	return addVoterFuture.Error()
}

func (a *RaftAgreementAdapter) Process(message r3model.R3Message) error {
	// TODO: manage non leader request handling
	if a.raft.State() != raft.Leader {
		return errors.New("not a raft leader")
	}
	buffer := bytes.NewBuffer([]byte{})
	err := gob.NewEncoder(buffer).Encode(message)
	if err != nil {
		return err
	}
	raftFuture := a.raft.Apply(buffer.Bytes(), a.proposalTimeout)
	return raftFuture.Error()
}

func (a *RaftAgreementAdapter) Ordered() <-chan r3model.R3Message {
	return a.orderedCh
}

////////////////////////////////////////////////////////////////////////////////
//
// raft.FSM interface implementation
//
////////////////////////////////////////////////////////////////////////////////

func (a *RaftAgreementAdapter) Apply(logEntry *raft.Log) interface{} {
	var message r3model.R3Message
	buffer := bytes.NewReader(logEntry.Data)
	err := gob.NewDecoder(buffer).Decode(&message)
	if err != nil {
		return err
	}
	a.orderedCh <- message
	return nil
}

func (a *RaftAgreementAdapter) Snapshot() (raft.FSMSnapshot, error) {
	// TODO: somehow get from proxy something to be used as snapshot
	return &raft.MockSnapshot{}, nil
}

func (a RaftAgreementAdapter) Restore(io.ReadCloser) error {
	return r3errors.NotImplementedYetError("RaftAgreementAdapter.Restore")
}
