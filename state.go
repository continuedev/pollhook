package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

const maxIDsPerSource = 10000

// State tracks seen IDs per source to provide at-least-once delivery dedup.
type State struct {
	mu      sync.Mutex
	dir     string
	sources map[string]*sourceState
}

type sourceState struct {
	IDs []string `json:"ids"`
	set map[string]bool
}

func newSourceState() *sourceState {
	return &sourceState{
		IDs: []string{},
		set: make(map[string]bool),
	}
}

func NewState(dir string) *State {
	return &State{
		dir:     dir,
		sources: make(map[string]*sourceState),
	}
}

// Load reads persisted state from disk.
func (s *State) Load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	path := filepath.Join(s.dir, "state.json")
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("reading state: %w", err)
	}

	var raw map[string][]string
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("parsing state: %w", err)
	}

	for name, ids := range raw {
		ss := newSourceState()
		for _, id := range ids {
			ss.IDs = append(ss.IDs, id)
			ss.set[id] = true
		}
		s.sources[name] = ss
	}
	return nil
}

// Save writes state to disk atomically (temp file + rename).
func (s *State) Save() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return fmt.Errorf("creating state dir: %w", err)
	}

	raw := make(map[string][]string)
	for name, ss := range s.sources {
		raw[name] = ss.IDs
	}

	data, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling state: %w", err)
	}

	path := filepath.Join(s.dir, "state.json")
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("writing temp state: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("renaming state: %w", err)
	}
	return nil
}

// HasID reports whether the given ID has been seen for the source.
func (s *State) HasID(source, id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	ss, ok := s.sources[source]
	if !ok {
		return false
	}
	return ss.set[id]
}

// AddID marks an ID as seen for the source. Applies sliding window cap.
func (s *State) AddID(source, id string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ss, ok := s.sources[source]
	if !ok {
		ss = newSourceState()
		s.sources[source] = ss
	}

	if ss.set[id] {
		return
	}

	ss.IDs = append(ss.IDs, id)
	ss.set[id] = true

	// Sliding window: drop oldest IDs if over cap
	if len(ss.IDs) > maxIDsPerSource {
		excess := len(ss.IDs) - maxIDsPerSource
		for _, old := range ss.IDs[:excess] {
			delete(ss.set, old)
		}
		ss.IDs = ss.IDs[excess:]
	}
}
