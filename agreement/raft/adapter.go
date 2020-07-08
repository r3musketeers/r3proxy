package raft

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"time"

	r3proxy "r3proxy/proxy"

	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb"
)

type RaftAgreementAdapter struct {
	nodeID          string
	address         string
	raft            *raft.Raft
	proposalTimeout time.Duration
	orderedCh       chan r3proxy.R3Message
	snapshot        func() *RaftSnapshot
	restore         func(*RaftSnapshot) error
}

func NewRaftAgreementAdapter(
	nodeID string,
	addrStr string,
	transportTimeout time.Duration,
	baseDir string,
	snapshotRetain int,
	proposalTimeout time.Duration,
	enableSingle bool,
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
		nodeID:          nodeID,
		address:         addrStr,
		proposalTimeout: proposalTimeout,
		orderedCh:       make(chan r3proxy.R3Message),
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

	if enableSingle {
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
// r3proxy.ConsensusAdapter interface implementation
//
////////////////////////////////////////////////////////////////////////////////

func (a *RaftAgreementAdapter) NodeID() string {
	return a.nodeID
}

func (a *RaftAgreementAdapter) Address() string {
	return a.address
}

func (a *RaftAgreementAdapter) SetHistoryGenerator(generateHistory r3proxy.HistoryGeneratorFunc) {
	a.snapshot = func() *RaftSnapshot {
		snapshot := RaftSnapshot(generateHistory())
		return &snapshot
	}
}

func (a *RaftAgreementAdapter) SetHistoryPopulator(populateHistory r3proxy.HistoryPopulatorFunc) {
	a.restore = func(snapshot *RaftSnapshot) error {
		populateHistory(map[string][]byte(*snapshot))
		return nil
	}
}

// func (a *RaftAgreementAdapter) Join(nodeID, addrStr string) error {
// 	log.Println("joining node", nodeID, addrStr)
// 	configFuture := a.raft.GetConfiguration()
// 	err := configFuture.Error()
// 	if err != nil {
// 		return err
// 	}
// 	for _, server := range configFuture.Configuration().Servers {
// 		if server.ID == raft.ServerID(nodeID) || server.Address == raft.ServerAddress(addrStr) {
// 			if server.ID == raft.ServerID(nodeID) && server.Address == raft.ServerAddress(addrStr) {
// 				log.Printf("node %s at %s already a member, ignoring request", nodeID, addrStr)
// 				return nil
// 			}
// 			removeFuture := a.raft.RemoveServer(server.ID, 0, 0)
// 			err = removeFuture.Error()
// 			if err != nil {
// 				return err
// 			}
// 		}
// 	}
// 	log.Println("adding voter")
// 	addVoterFuture := a.raft.AddVoter(raft.ServerID(nodeID), raft.ServerAddress(addrStr), 0, 0)
// 	err = addVoterFuture.Error()
// 	if err != nil {
// 		log.Println(err.Error())
// 		return err
// 	}
// 	return nil
// }

func (s *RaftAgreementAdapter) Join(nodeID, addr string) error {
	log.Printf("received join request for remote node %s at %s", nodeID, addr)

	configFuture := s.raft.GetConfiguration()
	if err := configFuture.Error(); err != nil {
		log.Printf("failed to get raft configuration: %v", err)
		return err
	}

	for _, srv := range configFuture.Configuration().Servers {
		// If a node already exists with either the joining node's ID or address,
		// that node may need to be removed from the config first.
		if srv.ID == raft.ServerID(nodeID) || srv.Address == raft.ServerAddress(addr) {
			// However if *both* the ID and the address are the same, then nothing -- not even
			// a join operation -- is needed.
			if srv.Address == raft.ServerAddress(addr) && srv.ID == raft.ServerID(nodeID) {
				log.Printf("node %s at %s already member of cluster, ignoring join request", nodeID, addr)
				return nil
			}

			future := s.raft.RemoveServer(srv.ID, 0, 0)
			if err := future.Error(); err != nil {
				return fmt.Errorf("error removing existing node %s at %s: %s", nodeID, addr, err)
			}
		}
	}

	f := s.raft.AddVoter(raft.ServerID(nodeID), raft.ServerAddress(addr), 0, 0)
	if f.Error() != nil {
		return f.Error()
	}
	log.Printf("node %s at %s joined successfully", nodeID, addr)
	return nil
}

func (a *RaftAgreementAdapter) Process(message r3proxy.R3Message) error {
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

func (a *RaftAgreementAdapter) Ordered() <-chan r3proxy.R3Message {
	return a.orderedCh
}

////////////////////////////////////////////////////////////////////////////////
//
// raft.FSM interface implementation
//
////////////////////////////////////////////////////////////////////////////////

func (a *RaftAgreementAdapter) Apply(logEntry *raft.Log) interface{} {
	// TODO: Add to local cache
	var message r3proxy.R3Message
	buffer := bytes.NewReader(logEntry.Data)
	err := gob.NewDecoder(buffer).Decode(&message)
	if err != nil {
		return err
	}
	a.orderedCh <- message
	return nil
}

func (a *RaftAgreementAdapter) Snapshot() (raft.FSMSnapshot, error) {
	return a.snapshot(), nil
}

func (a RaftAgreementAdapter) Restore(snapshotReader io.ReadCloser) error {
	snapshot := &RaftSnapshot{}
	err := gob.NewDecoder(snapshotReader).Decode(snapshot)
	if err != nil {
		return err
	}
	return a.restore(snapshot)
}
