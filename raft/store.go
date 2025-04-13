package raft

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/hashicorp/raft"
)

// Store provides an interface for storing and retrieving Raft data
type Store struct {
	mu sync.RWMutex
	// Path to the storage directory
	path string
	// Map to store values when not using persistence
	inMemory map[string][]byte
}

// NewStore creates a new store
func NewStore(path string) (*Store, error) {
	// Create the directory if it doesn't exist
	if path != "" {
		if err := os.MkdirAll(path, 0755); err != nil {
			return nil, fmt.Errorf("failed to create store directory: %v", err)
		}
	}

	return &Store{
		path:     path,
		inMemory: make(map[string][]byte),
	}, nil
}

// Set stores a key-value pair
func (s *Store) Set(key string, val []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// In-memory storage
	if s.path == "" {
		s.inMemory[key] = val
		return nil
	}

	// File-based storage
	path := filepath.Join(s.path, key)
	return os.WriteFile(path, val, 0644)
}

// Get retrieves a value by key
func (s *Store) Get(key string) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// In-memory storage
	if s.path == "" {
		val, ok := s.inMemory[key]
		if !ok {
			return nil, os.ErrNotExist
		}
		return val, nil
	}

	// File-based storage
	path := filepath.Join(s.path, key)
	return os.ReadFile(path)
}

// Delete removes a key-value pair
func (s *Store) Delete(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// In-memory storage
	if s.path == "" {
		delete(s.inMemory, key)
		return nil
	}

	// File-based storage
	path := filepath.Join(s.path, key)
	return os.Remove(path)
}

// Keys returns all keys in the store
func (s *Store) Keys() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var keys []string

	// In-memory storage
	if s.path == "" {
		for k := range s.inMemory {
			keys = append(keys, k)
		}
		return keys
	}

	// File-based storage
	files, err := os.ReadDir(s.path)
	if err != nil {
		return keys
	}

	for _, file := range files {
		if !file.IsDir() {
			keys = append(keys, file.Name())
		}
	}

	return keys
}

// StoreSnapshot stores a snapshot of the current state
func (s *Store) StoreSnapshot(data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// In-memory storage
	if s.path == "" {
		s.inMemory["snapshot"] = data
		return nil
	}

	// File-based storage
	path := filepath.Join(s.path, fmt.Sprintf("snapshot-%d", time.Now().Unix()))
	return os.WriteFile(path, data, 0644)
}

// LoadSnapshot loads the latest snapshot
func (s *Store) LoadSnapshot() ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// In-memory storage
	if s.path == "" {
		val, ok := s.inMemory["snapshot"]
		if !ok {
			return nil, os.ErrNotExist
		}
		return val, nil
	}

	// File-based storage - find the latest snapshot
	files, err := os.ReadDir(s.path)
	if err != nil {
		return nil, err
	}

	var latestSnapshot string
	var latestTime int64

	for _, file := range files {
		if !file.IsDir() && len(file.Name()) > 9 && file.Name()[:9] == "snapshot-" {
			// Extract timestamp from filename
			var timestamp int64
			_, err := fmt.Sscanf(file.Name(), "snapshot-%d", &timestamp)
			if err != nil {
				continue
			}

			if timestamp > latestTime {
				latestTime = timestamp
				latestSnapshot = file.Name()
			}
		}
	}

	if latestSnapshot == "" {
		return nil, os.ErrNotExist
	}

	return os.ReadFile(filepath.Join(s.path, latestSnapshot))
}

// PersistState persists the Raft server configuration
func (s *Store) PersistState(state raft.Configuration) error {
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	return s.Set("raft_state", data)
}

// LoadState loads the persisted Raft server configuration
func (s *Store) LoadState() (raft.Configuration, error) {
	data, err := s.Get("raft_state")
	if err != nil {
		if os.IsNotExist(err) {
			// Return empty configuration if not found
			return raft.Configuration{}, nil
		}
		return raft.Configuration{}, err
	}

	var state raft.Configuration
	if err := json.Unmarshal(data, &state); err != nil {
		return raft.Configuration{}, err
	}
	return state, nil
}

// Close closes the store
func (s *Store) Close() error {
	// Nothing to close for this implementation
	return nil
}

// Backup creates a backup of the store
func (s *Store) Backup(w io.Writer) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// In-memory storage
	if s.path == "" {
		return json.NewEncoder(w).Encode(s.inMemory)
	}

	// File-based storage
	files, err := os.ReadDir(s.path)
	if err != nil {
		return err
	}

	// Create a map to store all key-value pairs
	backup := make(map[string][]byte)

	for _, file := range files {
		if !file.IsDir() {
			data, err := os.ReadFile(filepath.Join(s.path, file.Name()))
			if err != nil {
				return err
			}
			backup[file.Name()] = data
		}
	}

	return json.NewEncoder(w).Encode(backup)
}

// Restore restores the store from a backup
func (s *Store) Restore(r io.Reader) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var backup map[string][]byte
	if err := json.NewDecoder(r).Decode(&backup); err != nil {
		return err
	}

	// In-memory storage
	if s.path == "" {
		s.inMemory = backup
		return nil
	}

	// File-based storage
	// First, clear the directory
	files, err := os.ReadDir(s.path)
	if err != nil {
		return err
	}

	for _, file := range files {
		if !file.IsDir() {
			if err := os.Remove(filepath.Join(s.path, file.Name())); err != nil {
				return err
			}
		}
	}

	// Then restore from backup
	for key, val := range backup {
		if err := os.WriteFile(filepath.Join(s.path, key), val, 0644); err != nil {
			return err
		}
	}

	return nil
}
