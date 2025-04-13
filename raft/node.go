package raft

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/devadigapratham/raft3d/api/models"
	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb/v2"
)

// Node represents a node in the Raft cluster
type Node struct {
	raft      *raft.Raft
	fsm       *FSM
	transport *raft.NetworkTransport
}

// Config represents the configuration for a Raft node
type Config struct {
	NodeID    string
	RaftAddr  string
	RaftDir   string
	Bootstrap bool
	Peers     []string
}

// NewNode creates a new Raft node
func NewNode(config *Config) (*Node, error) {
	// Create the FSM
	fsm := NewFSM()

	// Create Raft configuration
	raftConfig := raft.DefaultConfig()
	raftConfig.LocalID = raft.ServerID(config.NodeID)
	raftConfig.SnapshotInterval = 20 * time.Second
	raftConfig.SnapshotThreshold = 1024

	// Create the BoltDB store for logs
	logStorePath := filepath.Join(config.RaftDir, "raft-log.db")
	logStore, err := raftboltdb.NewBoltStore(logStorePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create BoltDB log store: %v", err)
	}

	// Create the stable store for data
	stableStorePath := filepath.Join(config.RaftDir, "raft-stable.db")
	stableStore, err := raftboltdb.NewBoltStore(stableStorePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create BoltDB stable store: %v", err)
	}

	// Create the snapshot store
	snapshotStore, err := raft.NewFileSnapshotStore(
		config.RaftDir, 3, os.Stderr)
	if err != nil {
		return nil, fmt.Errorf("failed to create snapshot store: %v", err)
	}

	// Setup TCP transport
	addr, err := net.ResolveTCPAddr("tcp", config.RaftAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve TCP address: %v", err)
	}
	transport, err := raft.NewTCPTransport(config.RaftAddr, addr, 3, 10*time.Second, os.Stderr)
	if err != nil {
		return nil, fmt.Errorf("failed to create TCP transport: %v", err)
	}

	// Create the Raft instance
	r, err := raft.NewRaft(
		raftConfig,
		fsm,
		logStore,
		stableStore,
		snapshotStore,
		transport,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Raft instance: %v", err)
	}

	// Bootstrap if needed
	if config.Bootstrap {
		// Create server configuration
		configuration := raft.Configuration{
			Servers: []raft.Server{
				{
					ID:      raft.ServerID(config.NodeID),
					Address: raft.ServerAddress(config.RaftAddr),
				},
			},
		}

		// Add other peers
		for _, peer := range config.Peers {
			if peer != config.RaftAddr {
				configuration.Servers = append(configuration.Servers, raft.Server{
					ID:      raft.ServerID(fmt.Sprintf("node-%s", peer)),
					Address: raft.ServerAddress(peer),
				})
			}
		}

		// Bootstrap the cluster
		f := r.BootstrapCluster(configuration)
		if err := f.Error(); err != nil && err != raft.ErrCantBootstrap {
			return nil, fmt.Errorf("failed to bootstrap cluster: %v", err)
		}
	}

	return &Node{
		raft:      r,
		fsm:       fsm,
		transport: transport,
	}, nil
}

// Apply applies a command to the Raft log
func (n *Node) Apply(cmd *models.Command) error {
	data, err := cmd.Marshal()
	if err != nil {
		return fmt.Errorf("failed to marshal command: %v", err)
	}

	// Apply the command to the Raft log
	future := n.raft.Apply(data, 5*time.Second)
	if err := future.Error(); err != nil {
		return fmt.Errorf("failed to apply command to Raft log: %v", err)
	}

	// Check for application error
	if appErr, ok := future.Response().(error); ok && appErr != nil {
		return fmt.Errorf("command application failed: %v", appErr)
	}

	return nil
}

// GetFSM returns the FSM
func (n *Node) GetFSM() *FSM {
	return n.fsm
}

// Leader returns true if this node is the leader
func (n *Node) Leader() bool {
	return n.raft.State() == raft.Leader
}

// LeaderAddress returns the address of the current leader
func (n *Node) LeaderAddress() string {
	return string(n.raft.Leader())
}

// State returns the current state of the Raft node
func (n *Node) State() raft.RaftState {
	return n.raft.State()
}

// Shutdown stops the Raft node
func (n *Node) Shutdown() error {
	// Shutdown the transport
	if n.transport != nil {
		n.transport.Close()
	}

	// Shutdown Raft
	if n.raft != nil {
		future := n.raft.Shutdown()
		return future.Error()
	}

	return nil
}
