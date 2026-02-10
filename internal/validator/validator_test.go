package validator

import (
	"strings"
	"testing"
)

func TestValidator_ValidServer(t *testing.T) {
	v, err := NewValidator()
	if err != nil {
		t.Fatalf("failed to create validator: %v", err)
	}

	valid := `{
		"name": "test/server",
		"description": "A test server",
		"version": "1.0.0"
	}`

	if err := v.ValidateServer([]byte(valid)); err != nil {
		t.Errorf("expected valid, got error: %v", err)
	}
}

func TestValidator_MissingRequired(t *testing.T) {
	v, err := NewValidator()
	if err != nil {
		t.Fatalf("failed to create validator: %v", err)
	}

	invalid := `{
		"name": "test/server"
	}`

	err = v.ValidateServer([]byte(invalid))
	if err == nil {
		t.Error("expected error for missing required fields")
	}

	if _, ok := err.(*ValidationError); !ok {
		t.Errorf("expected ValidationError, got %T", err)
	}
}

func TestValidator_InvalidNamePattern(t *testing.T) {
	v, err := NewValidator()
	if err != nil {
		t.Fatalf("failed to create validator: %v", err)
	}

	invalid := `{
		"name": "invalid-no-slash",
		"description": "Test",
		"version": "1.0.0"
	}`

	err = v.ValidateServer([]byte(invalid))
	if err == nil {
		t.Error("expected error for invalid name pattern")
	}
}

func TestValidator_InvalidJSON(t *testing.T) {
	v, err := NewValidator()
	if err != nil {
		t.Fatalf("failed to create validator: %v", err)
	}

	err = v.ValidateServer([]byte("not json"))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
	if !strings.Contains(err.Error(), "invalid JSON") {
		t.Errorf("expected 'invalid JSON' in error, got: %v", err)
	}
}
