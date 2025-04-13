package config

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

// Config represents the application configuration
type Config struct {
	// Node configuration
	NodeID    string
	RaftAddr  string
	RaftDir   string
	HTTPAddr  string
	Bootstrap bool
	JoinAddr  string
	Peers     []string
}

// ParseFlags parses command line flags and returns a Config
func ParseFlags() *Config {
	config := &Config{}

	// Define flags
	flag.StringVar(&config.NodeID, "id", "", "Node ID (required)")
	flag.StringVar(&config.RaftAddr, "raft-addr", "", "Raft transport address (required)")
	flag.StringVar(&config.RaftDir, "raft-dir", "", "Raft storage directory (required)")
	flag.StringVar(&config.HTTPAddr, "http-addr", "", "HTTP API address (required)")
	flag.BoolVar(&config.Bootstrap, "bootstrap", false, "Bootstrap the cluster")
	flag.StringVar(&config.JoinAddr, "join", "", "Join address of an existing node")
	peersStr := flag.String("peers", "", "Comma-separated list of peer addresses")

	// Parse flags
	flag.Parse()

	// Validate required flags
	if config.NodeID == "" {
		fmt.Fprintf(os.Stderr, "Node ID is required\n")
		flag.Usage()
		os.Exit(1)
	}

	if config.RaftAddr == "" {
		fmt.Fprintf(os.Stderr, "Raft address is required\n")
		flag.Usage()
		os.Exit(1)
	}

	if config.RaftDir == "" {
		fmt.Fprintf(os.Stderr, "Raft directory is required\n")
		flag.Usage()
		os.Exit(1)
	}

	if config.HTTPAddr == "" {
		fmt.Fprintf(os.Stderr, "HTTP address is required\n")
		flag.Usage()
		os.Exit(1)
	}

	// Parse peers
	if *peersStr != "" {
		config.Peers = strings.Split(*peersStr, ",")
	}

	return config
}
