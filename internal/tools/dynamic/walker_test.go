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
		if config.Metadata.Category == "bloom" {
			bloomToolsFound[config.Name] = true
			t.Logf("Found Bloom tool: %s (title: %s)", config.Name, config.Title)
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

func TestBloomToolsHaveRequiredFields(t *testing.T) {
	// Set the embedded FS
	EmbeddedFS = tools.ConfigFiles

	// Walk the config directory
	configs, err := WalkConfigDirectory("../../../tools/config")
	if err != nil {
		t.Fatalf("WalkConfigDirectory failed: %v", err)
	}

	// Check each Bloom tool has required fields
	for _, config := range configs {
		if config.Metadata.Category != "bloom" {
			continue
		}

		t.Logf("Validating Bloom tool: %s", config.Name)

		// Check required fields
		if config.Name == "" {
			t.Errorf("Bloom tool missing name")
		}
		if config.Title == "" {
			t.Errorf("Bloom tool %s missing title", config.Name)
		}
		if config.Description == "" {
			t.Errorf("Bloom tool %s missing description", config.Name)
		}

		// Bloom tools should be documentation-only (no execution block or input schema)
		if config.Execution != nil {
			t.Errorf("Bloom tool %s should not have execution block (documentation-only)", config.Name)
		}
		if config.InputSchema != nil {
			t.Errorf("Bloom tool %s should not have input schema (documentation-only)", config.Name)
		}

		// Check metadata
		if !config.Metadata.ReadOnly {
			t.Errorf("Bloom tool %s should be readonly", config.Name)
		}
		if !config.Metadata.Idempotent {
			t.Errorf("Bloom tool %s should be idempotent", config.Name)
		}
		if config.Metadata.Destructive {
			t.Errorf("Bloom tool %s should not be destructive", config.Name)
		}
	}
}
