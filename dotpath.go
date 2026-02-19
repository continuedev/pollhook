package main

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ExtractItems parses JSON data and navigates to the array at the given dot path.
// "." means the root is the array. "incidents" means $.incidents.
func ExtractItems(data []byte, path string) ([]json.RawMessage, error) {
	var parsed interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	target := parsed
	if path != "." {
		parts := strings.Split(path, ".")
		for _, part := range parts {
			m, ok := target.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("dot path %q: expected object, got %T", path, target)
			}
			next, exists := m[part]
			if !exists {
				return nil, fmt.Errorf("dot path %q: key %q not found", path, part)
			}
			target = next
		}
	}

	arr, ok := target.([]interface{})
	if !ok {
		return nil, fmt.Errorf("dot path %q: expected array, got %T", path, target)
	}

	items := make([]json.RawMessage, len(arr))
	for i, v := range arr {
		b, err := json.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("marshaling item %d: %w", i, err)
		}
		items[i] = b
	}
	return items, nil
}

// ExtractID extracts a string ID from a JSON object at the given dot path.
func ExtractID(item json.RawMessage, path string) (string, error) {
	var data interface{}
	if err := json.Unmarshal(item, &data); err != nil {
		return "", fmt.Errorf("invalid JSON: %w", err)
	}

	parts := strings.Split(path, ".")
	current := data
	for _, part := range parts {
		m, ok := current.(map[string]interface{})
		if !ok {
			return "", fmt.Errorf("id path %q: expected object, got %T", path, current)
		}
		next, exists := m[part]
		if !exists {
			return "", fmt.Errorf("id path %q: key %q not found", path, part)
		}
		current = next
	}

	return toString(current)
}

func toString(v interface{}) (string, error) {
	if v == nil {
		return "", fmt.Errorf("id is null")
	}
	switch val := v.(type) {
	case string:
		return val, nil
	case float64:
		if val == float64(int64(val)) {
			return fmt.Sprintf("%d", int64(val)), nil
		}
		return fmt.Sprintf("%g", val), nil
	case bool:
		return fmt.Sprintf("%t", val), nil
	default:
		return fmt.Sprintf("%v", val), nil
	}
}
