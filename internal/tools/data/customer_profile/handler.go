package customer_profile

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mkd-neo4j/neo4j-mcp-fraud/internal/tools"
	"github.com/mkd-neo4j/neo4j-mcp-fraud/internal/tools/cypher/query_builder"
)

// Handler returns the tool handler function for get-customer-profile
func Handler(deps *tools.ToolDependencies) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleGetCustomerProfile(ctx, request, deps)
	}
}

func handleGetCustomerProfile(ctx context.Context, request mcp.CallToolRequest, deps *tools.ToolDependencies) (*mcp.CallToolResult, error) {
	// Validate dependencies
	if deps.AnalyticsService == nil {
		errMessage := "Analytics service is not initialized"
		slog.Error(errMessage)
		return mcp.NewToolResultError(errMessage), nil
	}

	if deps.DBService == nil {
		errMessage := "Database service is not initialized"
		slog.Error(errMessage)
		return mcp.NewToolResultError(errMessage), nil
	}

	// Emit analytics event
	deps.AnalyticsService.EmitEvent(
		deps.AnalyticsService.NewToolsEvent("get-customer-profile"),
	)

	// Parse arguments
	var args GetCustomerProfileInput
	if err := request.BindArguments(&args); err != nil {
		slog.Error("error binding arguments", "error", err)
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Validate required parameters
	if args.EntityId == "" {
		errMessage := "entityId parameter is required"
		slog.Error(errMessage)
		return mcp.NewToolResultError(errMessage), nil
	}

	if args.EntityConfig.NodeLabel == "" {
		errMessage := "entityConfig.nodeLabel is required. Specify the entity node label (e.g., 'Customer', 'Person', 'Account')."
		slog.Error(errMessage)
		return mcp.NewToolResultError(errMessage), nil
	}

	if args.EntityConfig.IdProperty == "" {
		errMessage := "entityConfig.idProperty is required. Specify the property name containing the unique identifier (e.g., 'customerId', 'personId')."
		slog.Error(errMessage)
		return mcp.NewToolResultError(errMessage), nil
	}

	if len(args.AttributeMappings) == 0 {
		errMessage := "attributeMappings parameter is required and cannot be empty. Use get-schema to discover available attributes first."
		slog.Error(errMessage)
		return mcp.NewToolResultError(errMessage), nil
	}

	slog.Info("retrieving entity profile",
		"entityId", args.EntityId,
		"entityLabel", args.EntityConfig.NodeLabel,
		"attributeMappings", len(args.AttributeMappings))

	// Build dynamic Cypher query based on attribute mappings
	query := buildCustomerProfileQuery(args.EntityConfig, args.AttributeMappings)

	params := map[string]any{
		"entityId": args.EntityId,
	}

	slog.Debug("executing customer profile query", "query", query)

	// Execute query
	records, err := deps.DBService.ExecuteReadQuery(ctx, query, params)
	if err != nil {
		slog.Error("error executing customer profile query", "error", err)
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Format records to JSON
	response, err := deps.DBService.Neo4jRecordsToJSON(records)
	if err != nil {
		slog.Error("error formatting query results", "error", err)
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(response), nil
}

// buildCustomerProfileQuery constructs a dynamic Cypher query based on attribute mappings
func buildCustomerProfileQuery(entityConfig EntityConfig, mappings []query_builder.AttributeMapping) string {
	var queryBuilder strings.Builder

	// Start with base entity match using dynamic node label and ID property
	queryBuilder.WriteString(fmt.Sprintf("MATCH (e:%s {%s: $entityId})\n", entityConfig.NodeLabel, entityConfig.IdProperty))

	// Group mappings by category for organized output
	categorizedMappings := query_builder.GroupMappingsByCategory(mappings)

	// Build OPTIONAL MATCH clauses for each attribute
	matchBuilder := query_builder.NewOptionalMatchBuilder()
	varsByCategory := make(map[string][]string)

	for category, categoryMappings := range categorizedMappings {
		vars := make([]string, 0)
		for _, mapping := range categoryMappings {
			varName := matchBuilder.AddAttributeMatch("e", mapping)
			vars = append(vars, varName)
		}
		varsByCategory[category] = vars
	}

	// Add OPTIONAL MATCH clauses to query
	if matchBuilder.GetClauseCount() > 0 {
		queryBuilder.WriteString(matchBuilder.Build())
		queryBuilder.WriteString("\n")
	}

	// Build WITH clause to perform all aggregations
	// This allows RETURN to reference pre-aggregated variables without mixing with node properties
	queryBuilder.WriteString("WITH e")

	// Collect all attributes by category into pre-aggregated variables
	collectionAliases := make(map[string]map[string]string) // category -> {collectionKey -> alias}
	for category, categoryMappings := range categorizedMappings {
		collectionAliases[category] = make(map[string]string)
		for i, mapping := range categoryMappings {
			varName := varsByCategory[category][i]
			propMap := query_builder.BuildPropertyMap(varName, mapping)

			// Build collection key based on target label (pluralized, lowercase)
			collectionKey := strings.ToLower(mapping.TargetLabel) + "s"

			// Create unique alias for this collection
			collectionAlias := fmt.Sprintf("%s_%s", strings.ReplaceAll(category, "-", "_"), collectionKey)
			collectionAliases[category][collectionKey] = collectionAlias

			queryBuilder.WriteString(fmt.Sprintf(",\n     collect(DISTINCT %s) as %s", propMap, collectionAlias))
		}
	}
	queryBuilder.WriteString("\n")

	// Build RETURN clause - now only references pre-aggregated variables
	queryBuilder.WriteString("RETURN {\n")

	// Return base entity properties - safe to access node properties since no aggregation in RETURN
	queryBuilder.WriteString("  base_details: ")
	if len(entityConfig.BaseProperties) > 0 {
		// Build map from entity properties directly
		queryBuilder.WriteString("{\n")
		for i, prop := range entityConfig.BaseProperties {
			if i > 0 {
				queryBuilder.WriteString(",\n")
			}
			queryBuilder.WriteString(fmt.Sprintf("    %s: e.%s", prop, prop))
		}
		queryBuilder.WriteString("\n  }")
	} else {
		// Use properties map directly from entity
		queryBuilder.WriteString("properties(e)")
	}

	// Add collections for each category using pre-collected variables
	for category := range categorizedMappings {
		queryBuilder.WriteString(",\n")
		returnClause := buildCategoryReturnClauseFromCollections(category, collectionAliases[category])
		queryBuilder.WriteString(returnClause)
	}

	queryBuilder.WriteString("\n} as entityProfile")

	return queryBuilder.String()
}

// buildCategoryReturnClauseFromCollections constructs the RETURN clause using pre-collected variables
func buildCategoryReturnClauseFromCollections(category string, collectionAliases map[string]string) string {
	var clauseBuilder strings.Builder

	clauseBuilder.WriteString(fmt.Sprintf("  %s: {\n", category))

	i := 0
	for collectionKey, alias := range collectionAliases {
		if i > 0 {
			clauseBuilder.WriteString(",\n")
		}
		clauseBuilder.WriteString(fmt.Sprintf("    %s: %s", collectionKey, alias))
		i++
	}

	clauseBuilder.WriteString("\n  }")

	return clauseBuilder.String()
}
