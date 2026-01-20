package server

import (
	"log/slog"

	"github.com/mark3labs/mcp-go/server"
	"github.com/mkd-neo4j/neo4j-mcp-fraud/internal/tools"
	"github.com/mkd-neo4j/neo4j-mcp-fraud/internal/tools/cypher/read"
	"github.com/mkd-neo4j/neo4j-mcp-fraud/internal/tools/cypher/write"
	"github.com/mkd-neo4j/neo4j-mcp-fraud/internal/tools/dynamic"
	"github.com/mkd-neo4j/neo4j-mcp-fraud/internal/tools/gds"
	"github.com/mkd-neo4j/neo4j-mcp-fraud/internal/tools/models"
	"github.com/mkd-neo4j/neo4j-mcp-fraud/internal/tools/schema"
)

// registerTools registers all enabled MCP tools and adds them to the provided MCP server.
// Tools are filtered according to the server configuration. For example, when the read-only
// mode is enabled (e.g. via the NEO4J_READ_ONLY environment variable or the Config.ReadOnly flag),
// any tool that performs state mutation will be excluded; only tools annotated as read-only will be registered.
// Note: this read-only filtering relies on the tool annotation "readonly" (ReadOnlyHint). If the annotation
// is not defined or is set to false, the tool will be added (i.e., only tools with readonly=true are filtered in read-only mode).
func (s *Neo4jMCPServer) registerTools() error {
	filteredTools := s.getEnabledTools()
	s.MCPServer.AddTools(filteredTools...)
	return nil
}

type toolFilter func(tools []ToolDefinition) []ToolDefinition

type toolCategory int

const (
	cypherCategory  toolCategory = 0
	gdsCategory     toolCategory = 1
	fraudCategory   toolCategory = 2
	schemaCategory  toolCategory = 3
	dataCategory    toolCategory = 4 // Generic data retrieval tools
	dynamicCategory toolCategory = 5 // Dynamic config-based tools
)

type ToolDefinition struct {
	category   toolCategory
	definition server.ServerTool
	readonly   bool
}

func (s *Neo4jMCPServer) getEnabledTools() []server.ServerTool {
	filters := make([]toolFilter, 0)

	// If read-only mode is enabled, expose only tools annotated as read-only.
	if s.config != nil && s.config.ReadOnly {
		filters = append(filters, filterWriteTools)
	}
	// If GDS is not installed, disable GDS tools.
	if !s.gdsInstalled {
		filters = append(filters, filterGDSTools)
	}
	deps := &tools.ToolDependencies{
		DBService:        s.dbService,
		AnalyticsService: s.anService,
	}
	toolDefs := s.getAllToolsDefs(deps)

	for _, filter := range filters {
		toolDefs = filter(toolDefs)
	}
	enabledTools := make([]server.ServerTool, 0)
	for _, toolDef := range toolDefs {
		enabledTools = append(enabledTools, toolDef.definition)
	}
	return enabledTools
}

func filterWriteTools(tools []ToolDefinition) []ToolDefinition {
	readOnlyTools := make([]ToolDefinition, 0, len(tools))
	for _, t := range tools {
		if t.readonly {
			readOnlyTools = append(readOnlyTools, t)
		}
	}
	return readOnlyTools
}

func filterGDSTools(tools []ToolDefinition) []ToolDefinition {
	nonGDSTools := make([]ToolDefinition, 0, len(tools))
	for _, t := range tools {
		if t.category != gdsCategory {
			nonGDSTools = append(nonGDSTools, t)
		}
	}
	return nonGDSTools
}

// getAllToolsDefs returns all available tools with their specs and handlers
func (s *Neo4jMCPServer) getAllToolsDefs(deps *tools.ToolDependencies) []ToolDefinition {
	toolDefs := []ToolDefinition{
		{
			category: schemaCategory,
			definition: server.ServerTool{
				Tool:    schema.GetSchemaSpec(),
				Handler: schema.GetSchemaHandler(deps, s.config.SchemaSampleSize),
			},
			readonly: true,
		},
		{
			category: cypherCategory,
			definition: server.ServerTool{
				Tool:    read.ReadCypherSpec(),
				Handler: read.ReadCypherHandler(deps),
			},
			readonly: true,
		},
		{
			category: cypherCategory,
			definition: server.ServerTool{
				Tool:    write.WriteCypherSpec(),
				Handler: write.WriteCypherHandler(deps),
			},
			readonly: false,
		},
		// GDS Category/Section
		{
			category: gdsCategory,
			definition: server.ServerTool{
				Tool:    gds.ListGDSProceduresSpec(),
				Handler: gds.ListGdsProceduresHandler(deps),
			},
			readonly: true,
		},
		// Data Models Category/Section
		{
			category: schemaCategory,
			definition: server.ServerTool{
				Tool:    models.GetReferenceModelsSpec(),
				Handler: models.GetReferenceModelsHandler(deps),
			},
			readonly: true,
		},
		// Note: Data retrieval tools (get-customer-profile) are now config-based in tools/config/data/
	}

	// Load dynamic tools from config directory
	dynamicTools := s.loadDynamicTools(deps)
	toolDefs = append(toolDefs, dynamicTools...)

	return toolDefs
}

// loadDynamicTools loads tools from YAML configs in tools/config/ directory
func (s *Neo4jMCPServer) loadDynamicTools(deps *tools.ToolDependencies) []ToolDefinition {
	registry := dynamic.NewToolRegistry("tools/config")

	if err := registry.LoadTools(); err != nil {
		slog.Error("failed to load dynamic tools", "error", err)
		return []ToolDefinition{}
	}

	if registry.GetToolCount() == 0 {
		slog.Info("no dynamic tools found in config directory")
		return []ToolDefinition{}
	}

	slog.Info("loaded dynamic tools", "count", registry.GetToolCount())

	// Convert dynamic tools to ToolDefinition format
	serverTools := registry.GetServerTools(deps)
	toolDefs := make([]ToolDefinition, 0, len(serverTools))

	for _, serverTool := range serverTools {
		// All dynamic tools are categorized as dynamicCategory
		// Their specific category (fraud, data, etc.) is stored in metadata
		toolDef := ToolDefinition{
			category:   dynamicCategory,
			definition: serverTool,
			readonly:   true, // Dynamic tools specify readonly in their config
		}
		toolDefs = append(toolDefs, toolDef)
	}

	return toolDefs
}
