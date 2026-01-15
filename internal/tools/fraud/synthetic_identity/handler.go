package synthetic_identity

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mkd-neo4j/neo4j-mcp-fraud/internal/tools/fraud"
)

// Handler returns the tool handler function for synthetic identity fraud detection
func Handler(deps *fraud.ToolDeps) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleDetectSyntheticIdentity(ctx, request, deps)
	}
}

func handleDetectSyntheticIdentity(ctx context.Context, request mcp.CallToolRequest, deps *fraud.ToolDeps) (*mcp.CallToolResult, error) {
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
		deps.AnalyticsService.NewToolsEvent("detect-synthetic-identity"),
	)

	// Parse arguments
	var args DetectSyntheticIdentityInput
	if err := request.BindArguments(&args); err != nil {
		slog.Error("error binding arguments", "error", err)
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Validate required parameters
	if len(args.PIIRelationships) == 0 {
		errMessage := "piiRelationships parameter is required and cannot be empty. Use get-schema to discover available PII relationships first."
		slog.Error(errMessage)
		return mcp.NewToolResultError(errMessage), nil
	}

	if args.EntityConfig.NodeLabel == "" {
		errMessage := "entityConfig.nodeLabel is required. Specify the entity node label to search (e.g., 'Customer', 'Person', 'Account')."
		slog.Error(errMessage)
		return mcp.NewToolResultError(errMessage), nil
	}

	if args.EntityConfig.IdProperty == "" {
		errMessage := "entityConfig.idProperty is required. Specify the property name containing the unique identifier (e.g., 'customerId', 'personId')."
		slog.Error(errMessage)
		return mcp.NewToolResultError(errMessage), nil
	}

	// Set defaults
	minShared := args.MinSharedAttributes
	if minShared == 0 {
		minShared = 2
	}

	limit := args.Limit
	if limit == 0 {
		limit = 20
	}

	// Determine operation mode
	isInvestigationMode := args.EntityId != ""

	slog.Info("detecting synthetic identity fraud",
		"mode", map[bool]string{true: "investigation", false: "discovery"}[isInvestigationMode],
		"entityId", args.EntityId,
		"entityLabel", args.EntityConfig.NodeLabel,
		"piiRelationships", len(args.PIIRelationships),
		"minSharedAttributes", minShared,
		"limit", limit)

	// Build dynamic Cypher query based on mode and PII relationships
	var query string
	var params map[string]any

	if isInvestigationMode {
		// Investigation mode: find entities sharing PII with a specific entity
		query = buildInvestigationQuery(args.EntityConfig, args.PIIRelationships)
		params = map[string]any{
			"entityId":            args.EntityId,
			"minSharedAttributes": minShared,
			"limit":               limit,
		}
	} else {
		// Discovery mode: find all clusters of entities sharing PII
		query = buildDiscoveryQuery(args.EntityConfig, args.PIIRelationships)
		params = map[string]any{
			"minSharedAttributes": minShared,
			"limit":               limit,
		}
	}

	// Execute query
	records, err := deps.DBService.ExecuteReadQuery(ctx, query, params)
	if err != nil {
		slog.Error("error executing synthetic identity fraud query", "error", err)
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

// buildInvestigationQuery constructs a Cypher query for investigation mode (specific entity)
func buildInvestigationQuery(entityConfig EntityConfig, piiRelationships []PIIRelationship) string {
	relPattern, caseStatement := buildQueryComponents(piiRelationships)
	returnClause := buildReturnClause(entityConfig, "other")

	// Investigation mode: find entities sharing PII with a specific target entity
	query := fmt.Sprintf(`
		MATCH (target:%s {%s: $entityId})
		MATCH (target)-[r:%s]->(identifier)
		MATCH (identifier)<-[r2:%s]-(other:%s)
		WHERE target.%s <> other.%s
		WITH other,
		     collect(DISTINCT {
		         type: type(r2),
		         identifier: CASE
		             %s
		             ELSE 'Unknown'
		         END
		     }) as sharedAttributes
		WHERE size(sharedAttributes) >= $minSharedAttributes
		RETURN %s,
		       sharedAttributes,
		       size(sharedAttributes) as sharedAttributeCount
		ORDER BY sharedAttributeCount DESC
		LIMIT $limit
	`, entityConfig.NodeLabel, entityConfig.IdProperty,
		relPattern, relPattern, entityConfig.NodeLabel,
		entityConfig.IdProperty, entityConfig.IdProperty,
		caseStatement, returnClause)

	return query
}

// buildDiscoveryQuery constructs a Cypher query for discovery mode (find all clusters)
func buildDiscoveryQuery(entityConfig EntityConfig, piiRelationships []PIIRelationship) string {
	relPattern, caseStatement := buildQueryComponents(piiRelationships)
	returnClause1 := buildReturnClause(entityConfig, "e1")
	returnClause2 := buildReturnClause(entityConfig, "e2")

	// Discovery mode: find all pairs of entities sharing PII
	query := fmt.Sprintf(`
		MATCH (e1:%s)-[r1:%s]->(identifier)<-[r2:%s]-(e2:%s)
		WHERE id(e1) < id(e2)
		WITH e1, e2,
		     collect(DISTINCT {
		         type: type(r1),
		         identifier: CASE
		             %s
		             ELSE 'Unknown'
		         END
		     }) as sharedAttributes
		WHERE size(sharedAttributes) >= $minSharedAttributes
		WITH e1, e2, sharedAttributes, size(sharedAttributes) as sharedAttributeCount
		ORDER BY sharedAttributeCount DESC
		LIMIT $limit
		RETURN %s,
		       %s,
		       sharedAttributes,
		       sharedAttributeCount
	`, entityConfig.NodeLabel, relPattern, relPattern, entityConfig.NodeLabel,
		caseStatement, returnClause1, returnClause2)

	return query
}

// buildReturnClause builds the RETURN clause for entity properties
func buildReturnClause(entityConfig EntityConfig, varName string) string {
	// Always return the ID property
	returnParts := []string{
		fmt.Sprintf("%s.%s as %sId", varName, entityConfig.IdProperty, varName),
	}

	// Add display properties if specified
	if len(entityConfig.DisplayProperties) > 0 {
		for _, prop := range entityConfig.DisplayProperties {
			// Skip if this property is the same as the ID property (already included)
			if prop == entityConfig.IdProperty {
				continue
			}
			alias := fmt.Sprintf("%s%s", varName, titleCase(prop))
			returnParts = append(returnParts, fmt.Sprintf("%s.%s as %s", varName, prop, alias))
		}
	} else {
		// If no display properties specified, return all properties as a map
		returnParts = append(returnParts, fmt.Sprintf("properties(%s) as %sProperties", varName, varName))
	}

	return strings.Join(returnParts, ",\n               ")
}

// titleCase capitalizes the first letter of a string (replacement for deprecated strings.Title)
func titleCase(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// buildQueryComponents builds common query components (relationship pattern and CASE statement)
func buildQueryComponents(piiRelationships []PIIRelationship) (relPattern string, caseStatement string) {
	// Build the relationship type pattern (e.g., "HAS_EMAIL|HAS_PHONE|HAS_SSN")
	relTypes := make([]string, len(piiRelationships))
	for i, pii := range piiRelationships {
		relTypes[i] = pii.RelationshipType
	}
	relPattern = strings.Join(relTypes, "|")

	// Build the CASE statement for identifier extraction
	var caseClauses []string
	for _, pii := range piiRelationships {
		caseClauses = append(caseClauses,
			fmt.Sprintf("WHEN identifier:%s THEN identifier.%s", pii.TargetLabel, pii.IdentifierProperty))
	}
	caseStatement = strings.Join(caseClauses, "\n                 ")

	return relPattern, caseStatement
}
