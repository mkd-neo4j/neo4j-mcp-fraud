package dynamic

import (
	"fmt"
	"log/slog"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/mkd-neo4j/neo4j-mcp-fraud/internal/tools"
)

// ToolRegistry manages the loading and registration of dynamic tools
type ToolRegistry struct {
	configDir string
	configs   []*ToolConfig
}

// NewToolRegistry creates a new tool registry
func NewToolRegistry(configDir string) *ToolRegistry {
	return &ToolRegistry{
		configDir: configDir,
		configs:   make([]*ToolConfig, 0),
	}
}

// LoadTools loads all tool configurations from the config directory
func (r *ToolRegistry) LoadTools() error {
	configs, err := WalkConfigDirectory(r.configDir)
	if err != nil {
		return fmt.Errorf("failed to load tools from config directory: %w", err)
	}

	r.configs = configs
	slog.Info("loaded dynamic tools", "count", len(configs), "configDir", r.configDir)

	return nil
}

// GetToolCount returns the number of loaded tools
func (r *ToolRegistry) GetToolCount() int {
	return len(r.configs)
}

// GetTools returns all loaded tool configurations
func (r *ToolRegistry) GetTools() []*ToolConfig {
	return r.configs
}

// GetServerTools converts all loaded configs into MCP server tools
func (r *ToolRegistry) GetServerTools(deps *tools.ToolDependencies) []server.ServerTool {
	serverTools := make([]server.ServerTool, 0, len(r.configs))

	for _, config := range r.configs {
		tool := r.buildServerTool(config, deps)
		serverTools = append(serverTools, tool)
	}

	return serverTools
}

// buildServerTool creates an MCP server tool from a tool config
func (r *ToolRegistry) buildServerTool(config *ToolConfig, deps *tools.ToolDependencies) server.ServerTool {
	// Build enriched description from semantic fields
	description := buildEnrichedDescription(config)

	// Create the MCP tool specification
	// All config-based tools are guidance tools (readonly, idempotent, non-destructive)
	mcpTool := mcp.NewTool(config.Name,
		mcp.WithDescription(description),
		mcp.WithTitleAnnotation(config.Name), // Use name as title
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(true),
	)

	slog.Debug("built dynamic tool", "name", config.Name, "category", config.Category)

	// Create the handler
	handler := NewDynamicHandler(config, deps)

	return server.ServerTool{
		Tool:    mcpTool,
		Handler: handler,
	}
}

// GetCategory returns the category for a given tool name
func (r *ToolRegistry) GetCategory(toolName string) string {
	for _, config := range r.configs {
		if config.Name == toolName {
			return config.Category
		}
	}
	return "unknown"
}

// GetToolsByCategory returns all tools in a specific category
func (r *ToolRegistry) GetToolsByCategory(category string) []*ToolConfig {
	tools := make([]*ToolConfig, 0)
	for _, config := range r.configs {
		if config.Category == category {
			tools = append(tools, config)
		}
	}
	return tools
}

// ListCategories returns all unique categories
func (r *ToolRegistry) ListCategories() []string {
	categoryMap := make(map[string]bool)
	for _, config := range r.configs {
		categoryMap[config.Category] = true
	}

	categories := make([]string, 0, len(categoryMap))
	for category := range categoryMap {
		categories = append(categories, category)
	}

	return categories
}
