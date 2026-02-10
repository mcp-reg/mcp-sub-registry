package validator

import (
	"embed"
	"encoding/json"
	"fmt"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

//go:embed schemas/*.json
var schemaFS embed.FS

// Validator validates server.json files
type Validator struct {
	serverSchema *jsonschema.Schema
}

// NewValidator creates a new validator with embedded schemas
func NewValidator() (*Validator, error) {
	// Load the server schema
	schemaData, err := schemaFS.ReadFile("schemas/server.schema.json")
	if err != nil {
		return nil, fmt.Errorf("read schema: %w", err)
	}

	// Parse the schema JSON first
	var schemaDoc interface{}
	if err := json.Unmarshal(schemaData, &schemaDoc); err != nil {
		return nil, fmt.Errorf("parse schema json: %w", err)
	}

	c := jsonschema.NewCompiler()
	if err := c.AddResource("server.schema.json", schemaDoc); err != nil {
		return nil, fmt.Errorf("add schema resource: %w", err)
	}

	schema, err := c.Compile("server.schema.json")
	if err != nil {
		return nil, fmt.Errorf("compile schema: %w", err)
	}

	return &Validator{serverSchema: schema}, nil
}

// ValidateServer validates a server.json against the schema
func (v *Validator) ValidateServer(data []byte) error {
	var doc interface{}
	if err := json.Unmarshal(data, &doc); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	err := v.serverSchema.Validate(doc)
	if err != nil {
		return &ValidationError{Details: formatValidationError(err)}
	}

	return nil
}

// ValidationError represents a schema validation error
type ValidationError struct {
	Details string
}

func (e *ValidationError) Error() string {
	return "Invalid schema: " + e.Details
}

// formatValidationError extracts a readable error message
func formatValidationError(err error) string {
	if ve, ok := err.(*jsonschema.ValidationError); ok {
		// Get the first basic error for a concise message
		basics := ve.BasicOutput().Errors
		if len(basics) > 0 && basics[0].Error != nil {
			return basics[0].Error.String()
		}
	}
	return err.Error()
}
