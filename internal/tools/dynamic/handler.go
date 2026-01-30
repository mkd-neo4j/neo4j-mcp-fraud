package dynamic

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mkd-neo4j/neo4j-mcp-fraud/internal/tools"
)

// NewDynamicHandler creates a handler function for a dynamic tool
// All config-based tools return enriched descriptions as guidance for the LLM
func NewDynamicHandler(config *ToolConfig, deps *tools.ToolDependencies) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleDynamicTool(ctx, request, config, deps)
	}
}

func handleDynamicTool(ctx context.Context, request mcp.CallToolRequest, config *ToolConfig, deps *tools.ToolDependencies) (*mcp.CallToolResult, error) {
	// Emit analytics event if available
	if deps.AnalyticsService != nil {
		deps.AnalyticsService.EmitEvent(
			deps.AnalyticsService.NewToolsEvent(config.Name),
		)
	}

	slog.Info("guidance tool called", "tool", config.Name, "category", config.Category)

	// Build and return enriched description with all semantic fields
	enrichedDescription := buildEnrichedDescription(config)
	return mcp.NewToolResultText(enrichedDescription), nil
}

// buildEnrichedDescription creates a comprehensive description from all semantic fields
func buildEnrichedDescription(config *ToolConfig) string {
	var sb strings.Builder

	// Core description
	sb.WriteString(config.Description)

	// Intent - when to use this tool
	if config.Intent != "" {
		sb.WriteString("\n\n## Intent\n")
		sb.WriteString(config.Intent)
	}

	// Expected patterns - what this tool helps detect
	if len(config.ExpectedPatterns) > 0 {
		sb.WriteString("\n\n## Expected Patterns\n")
		for _, p := range config.ExpectedPatterns {
			sb.WriteString(fmt.Sprintf("- **%s**: %s\n", p.Entity, p.Anomaly))
			if len(p.SharedElements) > 0 {
				sb.WriteString(fmt.Sprintf("  Shared elements: %v\n", p.SharedElements))
			}
		}
	}

	// Reference Cypher - canonical implementation as guidance
	if config.ReferenceCypher != "" {
		sb.WriteString("\n\n## Reference Cypher\n```cypher\n")
		sb.WriteString(config.ReferenceCypher)
		sb.WriteString("\n```\n")
	}

	// Reference Schema - common labels and relationships to look for
	if config.ReferenceSchema != nil {
		sb.WriteString("\n\n## Reference Schema\n")
		if len(config.ReferenceSchema.Labels) > 0 {
			sb.WriteString(fmt.Sprintf("- Labels: %v\n", config.ReferenceSchema.Labels))
		}
		if len(config.ReferenceSchema.Relationships) > 0 {
			sb.WriteString(fmt.Sprintf("- Relationships: %v\n", config.ReferenceSchema.Relationships))
		}
	}

	// Parameters - expected query parameters
	if len(config.Parameters) > 0 {
		sb.WriteString("\n\n## Parameters\n")
		for _, p := range config.Parameters {
			sb.WriteString(fmt.Sprintf("- `$%s` (%s)", p.Name, p.Type))
			if p.Default != nil {
				sb.WriteString(fmt.Sprintf(" [default: %v]", p.Default))
			}
			if p.Description != "" {
				sb.WriteString(fmt.Sprintf(": %s", p.Description))
			}
			sb.WriteString("\n")
		}
	}

	return sb.String()
}
