package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestExtractItems_RootArray(t *testing.T) {
	data := `[{"id": 1}, {"id": 2}, {"id": 3}]`
	items, err := ExtractItems([]byte(data), ".")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}
}

func TestExtractItems_NestedPath(t *testing.T) {
	data := `{"data": {"incidents": [{"id": "a"}, {"id": "b"}]}}`
	items, err := ExtractItems([]byte(data), "data.incidents")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
}

func TestExtractItems_SingleKey(t *testing.T) {
	data := `{"results": [{"id": 10}]}`
	items, err := ExtractItems([]byte(data), "results")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
}

func TestExtractItems_NotArray(t *testing.T) {
	data := `{"results": "not an array"}`
	_, err := ExtractItems([]byte(data), "results")
	if err == nil {
		t.Fatal("expected error for non-array")
	}
}

func TestExtractItems_MissingKey(t *testing.T) {
	data := `{"other": [1]}`
	_, err := ExtractItems([]byte(data), "results")
	if err == nil {
		t.Fatal("expected error for missing key")
	}
}

func TestExtractID_String(t *testing.T) {
	item := json.RawMessage(`{"id": "abc-123"}`)
	id, err := ExtractID(item, "id")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "abc-123" {
		t.Fatalf("expected abc-123, got %s", id)
	}
}

func TestExtractID_Number(t *testing.T) {
	item := json.RawMessage(`{"id": 42}`)
	id, err := ExtractID(item, "id")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "42" {
		t.Fatalf("expected 42, got %s", id)
	}
}

func TestExtractID_Nested(t *testing.T) {
	item := json.RawMessage(`{"meta": {"uid": "xyz"}}`)
	id, err := ExtractID(item, "meta.uid")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "xyz" {
		t.Fatalf("expected xyz, got %s", id)
	}
}

func TestExtractID_Missing(t *testing.T) {
	item := json.RawMessage(`{"name": "test"}`)
	_, err := ExtractID(item, "id")
	if err == nil {
		t.Fatal("expected error for missing id")
	}
}

func TestState_HasAddID(t *testing.T) {
	dir := t.TempDir()
	s := NewState(dir)

	if s.HasID("src", "1") {
		t.Fatal("expected false for unseen ID")
	}

	s.AddID("src", "1")
	if !s.HasID("src", "1") {
		t.Fatal("expected true after AddID")
	}

	// Adding same ID again should be a no-op
	s.AddID("src", "1")
	s.mu.Lock()
	count := len(s.sources["src"].IDs)
	s.mu.Unlock()
	if count != 1 {
		t.Fatalf("expected 1 ID after duplicate add, got %d", count)
	}
}

func TestState_SlidingWindowCap(t *testing.T) {
	dir := t.TempDir()
	s := NewState(dir)

	// Add maxIDsPerSource + 100 IDs
	for i := 0; i < maxIDsPerSource+100; i++ {
		s.AddID("src", fmt.Sprintf("id-%d", i))
	}

	s.mu.Lock()
	count := len(s.sources["src"].IDs)
	s.mu.Unlock()
	if count != maxIDsPerSource {
		t.Fatalf("expected %d IDs after cap, got %d", maxIDsPerSource, count)
	}
}

func TestState_PersistAndLoad(t *testing.T) {
	dir := t.TempDir()

	// Save state
	s1 := NewState(dir)
	s1.AddID("src1", "a")
	s1.AddID("src1", "b")
	s1.AddID("src2", "x")
	if err := s1.Save(); err != nil {
		t.Fatalf("save error: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(filepath.Join(dir, "state.json")); err != nil {
		t.Fatalf("state file not created: %v", err)
	}

	// Load into new state
	s2 := NewState(dir)
	if err := s2.Load(); err != nil {
		t.Fatalf("load error: %v", err)
	}

	if !s2.HasID("src1", "a") || !s2.HasID("src1", "b") {
		t.Fatal("expected persisted IDs for src1")
	}
	if !s2.HasID("src2", "x") {
		t.Fatal("expected persisted IDs for src2")
	}
	if s2.HasID("src1", "c") {
		t.Fatal("unexpected ID found")
	}
}

func TestState_LoadMissing(t *testing.T) {
	dir := t.TempDir()
	s := NewState(dir)
	if err := s.Load(); err != nil {
		t.Fatalf("load of missing file should not error: %v", err)
	}
}

func TestConfig_Valid(t *testing.T) {
	yaml := `
sources:
  - name: test
    command: echo '[]'
    interval: 5m
    items: "."
    id: "id"
    webhook:
      url: https://example.com/hook
      secret: mysecret
`
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	os.WriteFile(path, []byte(yaml), 0o644)

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Sources) != 1 {
		t.Fatalf("expected 1 source, got %d", len(cfg.Sources))
	}
	if cfg.Sources[0].Name != "test" {
		t.Fatalf("expected name 'test', got %q", cfg.Sources[0].Name)
	}
	if cfg.Sources[0].Webhook.Secret != "mysecret" {
		t.Fatalf("expected secret 'mysecret', got %q", cfg.Sources[0].Webhook.Secret)
	}
}

func TestConfig_MissingName(t *testing.T) {
	yaml := `
sources:
  - command: echo '[]'
    interval: 5m
    items: "."
    id: "id"
    webhook:
      url: https://example.com/hook
`
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	os.WriteFile(path, []byte(yaml), 0o644)

	_, err := LoadConfig(path)
	if err == nil {
		t.Fatal("expected validation error for missing name")
	}
}

func TestConfig_DuplicateName(t *testing.T) {
	yaml := `
sources:
  - name: dupe
    command: echo '[]'
    interval: 5m
    items: "."
    id: "id"
    webhook:
      url: https://example.com/hook
  - name: dupe
    command: echo '[]'
    interval: 5m
    items: "."
    id: "id"
    webhook:
      url: https://example.com/hook
`
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	os.WriteFile(path, []byte(yaml), 0o644)

	_, err := LoadConfig(path)
	if err == nil {
		t.Fatal("expected validation error for duplicate name")
	}
}

func TestConfig_IntervalTooShort(t *testing.T) {
	yaml := `
sources:
  - name: test
    command: echo '[]'
    interval: 500ms
    items: "."
    id: "id"
    webhook:
      url: https://example.com/hook
`
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	os.WriteFile(path, []byte(yaml), 0o644)

	_, err := LoadConfig(path)
	if err == nil {
		t.Fatal("expected validation error for short interval")
	}
}

func TestConfig_NoSources(t *testing.T) {
	yaml := `sources: []`
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	os.WriteFile(path, []byte(yaml), 0o644)

	_, err := LoadConfig(path)
	if err == nil {
		t.Fatal("expected validation error for empty sources")
	}
}
