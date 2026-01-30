package dynamic

import (
	"testing"

	"github.com/mkd-neo4j/neo4j-mcp-fraud/tools"
)

func TestWalkConfigDirectory_IncludesBloomTools(t *testing.T) {
	// Set the embedded FS
	EmbeddedFS = tools.ConfigFiles

	// Walk the config directory
	configs, err := WalkConfigDirectory("../../../tools/config")
	if err != nil {
		t.Fatalf("WalkConfigDirectory failed: %v", err)
	}

	// Check for Bloom tools
	bloomToolsFound := make(map[string]bool)
	bloomTools := []string{
		"generate-scene-action",
		"generate-search-phrase",
	}

	for _, config := range configs {
		if config.Category == "bloom" {
			bloomToolsFound[config.Name] = true
			t.Logf("Found Bloom tool: %s", config.Name)
		}
	}

	// Verify both Bloom tools are discovered
	for _, toolName := range bloomTools {
		if !bloomToolsFound[toolName] {
			t.Errorf("Expected Bloom tool %s not found", toolName)
		}
	}

	// Verify we found at least 2 Bloom tools
	if len(bloomToolsFound) < 2 {
		t.Errorf("Expected at least 2 Bloom tools, found %d", len(bloomToolsFound))
	}
}

func TestToolsHaveRequiredFields(t *testing.T) {
	// Set the embedded FS
	EmbeddedFS = tools.ConfigFiles

	// Walk the config directory
	configs, err := WalkConfigDirectory("../../../tools/config")
	if err != nil {
		t.Fatalf("WalkConfigDirectory failed: %v", err)
	}

	// Check each tool has required fields
	for _, config := range configs {
		t.Logf("Validating tool: %s (category: %s)", config.Name, config.Category)

		// Check required fields
		if config.Name == "" {
			t.Errorf("Tool missing name")
		}
		if config.Description == "" {
			t.Errorf("Tool %s missing description", config.Name)
		}
		if config.Category == "" {
			t.Errorf("Tool %s missing category", config.Name)
		}
	}
}

func TestValidateParameters(t *testing.T) {
	tests := []struct {
		name    string
		params  []ParameterConfig
		wantErr bool
	}{
		{
			name:    "empty params is valid",
			params:  []ParameterConfig{},
			wantErr: false,
		},
		{
			name: "valid params",
			params: []ParameterConfig{
				{Name: "time_window_days", Type: "integer", Default: 90},
				{Name: "min_shared", Type: "integer", Default: 2},
			},
			wantErr: false,
		},
		{
			name: "missing name is invalid",
			params: []ParameterConfig{
				{Type: "integer"},
			},
			wantErr: true,
		},
		{
			name: "duplicate name is invalid",
			params: []ParameterConfig{
				{Name: "foo", Type: "string"},
				{Name: "foo", Type: "integer"},
			},
			wantErr: true,
		},
		{
			name: "invalid type is invalid",
			params: []ParameterConfig{
				{Name: "foo", Type: "invalid_type"},
			},
			wantErr: true,
		},
		{
			name: "empty type is valid (optional)",
			params: []ParameterConfig{
				{Name: "foo"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateParameters(tt.params)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateParameters() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
