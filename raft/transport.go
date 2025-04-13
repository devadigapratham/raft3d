package raft

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/hashicorp/raft"
)

// Transport provides methods for forwarding requests to the Raft leader
type Transport struct {
	node *Node
}

// NewTransport creates a new Transport
func NewTransport(node *Node) *Transport {
	return &Transport{
		node: node,
	}
}

// ForwardToLeader forwards a request to the Raft leader
func (t *Transport) ForwardToLeader(method, path string, body []byte) ([]byte, error) {
	// If this node is the leader, no need to forward
	if t.node.Leader() {
		return nil, nil
	}

	// Get the leader's address
	leaderAddr := t.node.LeaderAddress()
	if leaderAddr == "" {
		return nil, fmt.Errorf("no leader available")
	}

	// Extract HTTP address from raft address (this assumes a convention where Raft port and HTTP port have a fixed relationship)
	httpPort := 8000
	raftPort := 7000
	leaderPort := 0
	_, err := fmt.Sscanf(leaderAddr[len(leaderAddr)-4:], "%d", &leaderPort)
	if err != nil {
		return nil, fmt.Errorf("failed to parse leader port: %v", err)
	}

	// Calculate HTTP port from Raft port
	leaderHTTPPort := httpPort + (leaderPort - raftPort)
	leaderHTTPAddr := fmt.Sprintf("http://localhost:%d", leaderHTTPPort)

	// Create the request
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, method, leaderHTTPAddr+path, nil)
	if err != nil {
		return nil, err
	}

	// Set the body if provided
	if body != nil {
		req.Body = http.NoBody
		req.Header.Set("Content-Type", "application/json")
	}

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Check the response
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("received non-success response: %d", resp.StatusCode)
	}

	// Read the response
	var respBody []byte
	if resp.ContentLength > 0 {
		respBody = make([]byte, resp.ContentLength)
		_, err = resp.Body.Read(respBody)
		if err != nil {
			return nil, err
		}
	}

	return respBody, nil
}

// JoinCluster joins a node to the Raft cluster
func (t *Transport) JoinCluster(nodeID, nodeAddr string) error {
	// Prepare the request body
	body, err := json.Marshal(map[string]string{
		"node_id":   nodeID,
		"node_addr": nodeAddr,
	})
	if err != nil {
		return err
	}

	// Forward to leader
	_, err = t.ForwardToLeader("POST", "/join", body)
	return err
}

// LeaveCluster removes a node from the Raft cluster
func (t *Transport) LeaveCluster(nodeID string) error {
	// Prepare the request body
	body, err := json.Marshal(map[string]string{
		"node_id": nodeID,
	})
	if err != nil {
		return err
	}

	// Forward to leader
	_, err = t.ForwardToLeader("POST", "/leave", body)
	return err
}

// RaftHandler returns an HTTP handler for Raft-related operations
func (t *Transport) RaftHandler() http.Handler {
	mux := http.NewServeMux()

	// Handler for joining the cluster
	mux.HandleFunc("/join", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Only the leader can add nodes
		if !t.node.Leader() {
			http.Error(w, "Not the leader", http.StatusConflict)
			return
		}

		// Parse the request
		var req struct {
			NodeID   string `json:"node_id"`
			NodeAddr string `json:"node_addr"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, fmt.Sprintf("Failed to decode request: %v", err), http.StatusBadRequest)
			return
		}

		// Add the node to the Raft cluster
		future := t.node.raft.AddVoter(
			raft.ServerID(req.NodeID),
			raft.ServerAddress(req.NodeAddr),
			0,
			0,
		)

		if err := future.Error(); err != nil {
			http.Error(w, fmt.Sprintf("Failed to add node: %v", err), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	})

	// Handler for leaving the cluster
	mux.HandleFunc("/leave", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Only the leader can remove nodes
		if !t.node.Leader() {
			http.Error(w, "Not the leader", http.StatusConflict)
			return
		}

		// Parse the request
		var req struct {
			NodeID string `json:"node_id"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, fmt.Sprintf("Failed to decode request: %v", err), http.StatusBadRequest)
			return
		}

		// Remove the node from the Raft cluster
		future := t.node.raft.RemoveServer(raft.ServerID(req.NodeID), 0, 0)
		if err := future.Error(); err != nil {
			http.Error(w, fmt.Sprintf("Failed to remove node: %v", err), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	})

	return mux
}
