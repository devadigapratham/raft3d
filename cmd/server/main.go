package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/devadigapratham/raft3d/api"
	"github.com/devadigapratham/raft3d/config"
	"github.com/devadigapratham/raft3d/raft"
)

func main() {
	// Parse command line flags
	cfg := config.ParseFlags()

	// Create Raft data directory if it doesn't exist
	if err := os.MkdirAll(cfg.RaftDir, 0755); err != nil {
		log.Fatalf("Failed to create Raft directory: %v", err)
	}

	// Create a unique node ID if not provided
	if cfg.NodeID == "" {
		cfg.NodeID = filepath.Base(cfg.RaftDir)
	}

	// Create the store
	store, err := raft.NewStore(filepath.Join(cfg.RaftDir, "store"))
	if err != nil {
		log.Fatalf("Failed to create store: %v", err)
	}

	// Create Raft node
	raftConfig := &raft.Config{
		NodeID:    cfg.NodeID,
		RaftAddr:  cfg.RaftAddr,
		RaftDir:   cfg.RaftDir,
		Bootstrap: cfg.Bootstrap,
		Peers:     cfg.Peers,
	}

	node, err := raft.NewNode(raftConfig)
	if err != nil {
		log.Fatalf("Failed to create Raft node: %v", err)
	}

	// Create transport
	transport := raft.NewTransport(node)

	// Setup HTTP router
	router := api.SetupRouter(node)

	// Add Raft transport handler
	http.Handle("/raft/", http.StripPrefix("/raft", transport.RaftHandler()))

	// Start HTTP server
	server := &http.Server{
		Addr:    cfg.HTTPAddr,
		Handler: router,
	}

	// Start the server in a goroutine
	go func() {
		log.Printf("Starting HTTP server on %s", cfg.HTTPAddr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start HTTP server: %v", err)
		}
	}()

	// Join the cluster if needed
	if cfg.JoinAddr != "" && !cfg.Bootstrap {
		log.Printf("Joining cluster at %s", cfg.JoinAddr)
		if err := transport.JoinCluster(cfg.NodeID, cfg.RaftAddr); err != nil {
			log.Printf("Failed to join cluster: %v", err)
			// Continue anyway, as this is not critical
		}
	}

	// Handle shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down...")

	// Close the store
	if err := store.Close(); err != nil {
		log.Printf("Error closing store: %v", err)
	}

	// Shutdown Raft node
	if err := node.Shutdown(); err != nil {
		log.Printf("Error shutting down Raft node: %v", err)
	}

	log.Println("Shutdown complete")
}
