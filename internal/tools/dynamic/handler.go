package dynamic

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mkd-neo4j/neo4j-mcp-fraud/internal/tools"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// DynamicToolInput represents the generic input format for all dynamic tools
type DynamicToolInput struct {
	// Query is the Cypher query string (required)
	Query string `json:"query"`

	// Params contains optional query parameters
	Params map[string]interface{} `json:"params,omitempty"`
}

// NewDynamicHandler creates a handler function for a dynamic tool
func NewDynamicHandler(config *ToolConfig, deps *tools.ToolDependencies) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleDynamicTool(ctx, request, config, deps)
	}
}

func handleDynamicTool(ctx context.Context, request mcp.CallToolRequest, config *ToolConfig, deps *tools.ToolDependencies) (*mcp.CallToolResult, error) {
	// Validate dependencies
	if deps.AnalyticsService == nil {
		errMessage := "Analytics service is not initialized"
		slog.Error(errMessage)
		return mcp.NewToolResultError(errMessage), nil
	}

	// Emit analytics event
	deps.AnalyticsService.EmitEvent(
		deps.AnalyticsService.NewToolsEvent(config.Name),
	)

	// Check if this is a documentation tool (no execution block)
	if config.Execution == nil {
		slog.Info("documentation tool called", "tool", config.Name, "category", config.Metadata.Category)

		// For documentation tools, return the description as the content
		// The description field contains the full documentation/guidance
		return mcp.NewToolResultText(config.Description), nil
	}

	if deps.DBService == nil {
		errMessage := "Database service is not initialized"
		slog.Error(errMessage)
		return mcp.NewToolResultError(errMessage), nil
	}

	// Parse arguments for query-based tools
	var args DynamicToolInput
	if err := request.BindArguments(&args); err != nil {
		slog.Error("error binding arguments", "tool", config.Name, "error", err)
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Validate required fields
	if args.Query == "" {
		errMessage := "query parameter is required"
		slog.Error(errMessage, "tool", config.Name)
		return mcp.NewToolResultError(errMessage), nil
	}

	// Security validation: check if query matches tool's execution mode
	if err := validateQueryMode(args.Query, config.Execution.Mode); err != nil {
		slog.Error("query validation failed", "tool", config.Name, "error", err)
		return mcp.NewToolResultError(err.Error()), nil
	}

	slog.Info("executing dynamic tool",
		"tool", config.Name,
		"category", config.Metadata.Category,
		"mode", config.Execution.Mode,
		"hasParams", len(args.Params) > 0)

	// Execute query based on mode
	var records []*neo4j.Record
	var err error

	if config.Execution.Mode == "read" {
		records, err = deps.DBService.ExecuteReadQuery(ctx, args.Query, args.Params)
	} else {
		records, err = deps.DBService.ExecuteWriteQuery(ctx, args.Query, args.Params)
	}

	if err != nil {
		slog.Error("error executing query", "tool", config.Name, "error", err)
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Format records to JSON
	response, err := deps.DBService.Neo4jRecordsToJSON(records)
	if err != nil {
		slog.Error("error formatting query results", "tool", config.Name, "error", err)
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(response), nil
}

// validateQueryMode checks if the query matches the declared execution mode
// This is a basic security check to prevent write queries in read-only tools
func validateQueryMode(query string, mode string) error {
	normalizedQuery := strings.ToUpper(strings.TrimSpace(query))

	if mode == "read" {
		// Check for write operations in read mode
		writeKeywords := []string{
			"CREATE ", "MERGE ", "DELETE ", "REMOVE ", "SET ",
			"DROP ", "DETACH DELETE", "CALL {", // CALL with subqueries can be write
		}

		for _, keyword := range writeKeywords {
			if strings.Contains(normalizedQuery, keyword) {
				return fmt.Errorf("write operation detected in read-only tool: %s", keyword)
			}
		}
	}

	return nil
}
