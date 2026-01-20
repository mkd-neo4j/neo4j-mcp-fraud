package server

import (
	"os"
	"path/filepath"
	"testing"

	analytics_mocks "github.com/mkd-neo4j/neo4j-mcp-fraud/internal/analytics/mocks"
	"github.com/mkd-neo4j/neo4j-mcp-fraud/internal/config"
	database_mocks "github.com/mkd-neo4j/neo4j-mcp-fraud/internal/database/mocks"
	"github.com/mkd-neo4j/neo4j-mcp-fraud/internal/tools"
	"go.uber.org/mock/gomock"
)

func getProjectRoot(t *testing.T) string {
	// Start from current directory and walk up until we find go.mod
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("Could not find project root (go.mod not found)")
		}
		dir = parent
	}
}

func TestDynamicToolsAreExposed(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Change to project root so relative paths work
	projectRoot := getProjectRoot(t)
	oldDir, _ := os.Getwd()
	if err := os.Chdir(projectRoot); err != nil {
		t.Fatalf("Failed to change to project root: %v", err)
	}
	defer os.Chdir(oldDir)

	// Create a minimal server instance
	cfg := &config.Config{
		ReadOnly: false,
	}

	server := &Neo4jMCPServer{
		config:       cfg,
		dbService:    database_mocks.NewMockService(ctrl),
		anService:    analytics_mocks.NewMockService(ctrl),
		gdsInstalled: false,
	}

	// Get all tool definitions
	deps := &tools.ToolDependencies{
		DBService:        server.dbService,
		AnalyticsService: server.anService,
	}
	toolDefs := server.getAllToolsDefs(deps)

	// Check that we have tools
	if len(toolDefs) == 0 {
		t.Fatal("No tools found")
	}

	// Count dynamic tools
	dynamicCount := 0
	var dynamicToolNames []string

	for _, toolDef := range toolDefs {
		if toolDef.category == dynamicCategory {
			dynamicCount++
			dynamicToolNames = append(dynamicToolNames, toolDef.definition.Tool.Name)
		}
	}

	t.Logf("Total tools: %d", len(toolDefs))
	t.Logf("Dynamic tools: %d", dynamicCount)
	t.Logf("Dynamic tool names: %v", dynamicToolNames)

	// Verify we have the expected dynamic tools
	expectedTools := map[string]bool{
		"detect-synthetic-identity": false,
		"get-customer-profile":      false,
		"get-sar-report-guidance":   false,
	}

	for _, name := range dynamicToolNames {
		if _, exists := expectedTools[name]; exists {
			expectedTools[name] = true
		}
	}

	// Check all expected tools were found
	for toolName, found := range expectedTools {
		if !found {
			t.Errorf("Expected dynamic tool not found: %s", toolName)
		}
	}

	// Verify minimum dynamic tool count
	if dynamicCount < 3 {
		t.Errorf("Expected at least 3 dynamic tools, got %d", dynamicCount)
	}
}

func TestDynamicToolsHaveCorrectStructure(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Change to project root so relative paths work
	projectRoot := getProjectRoot(t)
	oldDir, _ := os.Getwd()
	if err := os.Chdir(projectRoot); err != nil {
		t.Fatalf("Failed to change to project root: %v", err)
	}
	defer os.Chdir(oldDir)

	cfg := &config.Config{
		ReadOnly: false,
	}

	server := &Neo4jMCPServer{
		config:       cfg,
		dbService:    database_mocks.NewMockService(ctrl),
		anService:    analytics_mocks.NewMockService(ctrl),
		gdsInstalled: false,
	}

	deps := &tools.ToolDependencies{
		DBService:        server.dbService,
		AnalyticsService: server.anService,
	}
	toolDefs := server.getAllToolsDefs(deps)

	for _, toolDef := range toolDefs {
		if toolDef.category != dynamicCategory {
			continue
		}

		tool := toolDef.definition.Tool
		t.Logf("Checking tool: %s", tool.Name)

		// Verify tool has required fields
		if tool.Name == "" {
			t.Errorf("Tool has empty name")
		}

		if tool.Description == "" {
			t.Errorf("Tool %s has empty description", tool.Name)
		}

		// Verify handler is not nil
		if toolDef.definition.Handler == nil {
			t.Errorf("Tool %s has nil handler", tool.Name)
		}

		// All dynamic tools should be readonly
		if !toolDef.readonly {
			t.Errorf("Tool %s is not marked as readonly", tool.Name)
		}
	}
}
