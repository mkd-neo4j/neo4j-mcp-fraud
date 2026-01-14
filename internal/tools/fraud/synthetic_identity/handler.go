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
	isInvestigationMode := args.CustomerId != ""

	slog.Info("detecting synthetic identity fraud",
		"mode", map[bool]string{true: "investigation", false: "discovery"}[isInvestigationMode],
		"customerId", args.CustomerId,
		"piiRelationships", len(args.PIIRelationships),
		"minSharedAttributes", minShared,
		"limit", limit)

	// Build dynamic Cypher query based on mode and PII relationships
	var query string
	var params map[string]any

	if isInvestigationMode {
		// Investigation mode: find customers sharing PII with a specific customer
		query = buildInvestigationQuery(args.PIIRelationships)
		params = map[string]any{
			"customerId":          args.CustomerId,
			"minSharedAttributes": minShared,
			"limit":               limit,
		}
	} else {
		// Discovery mode: find all clusters of customers sharing PII
		query = buildDiscoveryQuery(args.PIIRelationships)
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

// buildInvestigationQuery constructs a Cypher query for investigation mode (specific customer)
func buildInvestigationQuery(piiRelationships []PIIRelationship) string {
	relPattern, caseStatement := buildQueryComponents(piiRelationships)

	// Investigation mode: find customers sharing PII with a specific target customer
	query := fmt.Sprintf(`
		MATCH (target:Customer {customerId: $customerId})
		MATCH (target)-[r:%s]->(identifier)
		MATCH (identifier)<-[r2:%s]-(other:Customer)
		WHERE target.customerId <> other.customerId
		WITH other,
		     collect(DISTINCT {
		         type: type(r2),
		         identifier: CASE
		             %s
		             ELSE 'Unknown'
		         END
		     }) as sharedAttributes
		WHERE size(sharedAttributes) >= $minSharedAttributes
		RETURN other.customerId as customerId,
		       other.firstName as firstName,
		       other.lastName as lastName,
		       sharedAttributes,
		       size(sharedAttributes) as sharedAttributeCount
		ORDER BY sharedAttributeCount DESC
		LIMIT $limit
	`, relPattern, relPattern, caseStatement)

	return query
}

// buildDiscoveryQuery constructs a Cypher query for discovery mode (find all clusters)
func buildDiscoveryQuery(piiRelationships []PIIRelationship) string {
	relPattern, caseStatement := buildQueryComponents(piiRelationships)

	// Discovery mode: find all pairs of customers sharing PII
	query := fmt.Sprintf(`
		MATCH (c1:Customer)-[r1:%s]->(identifier)<-[r2:%s]-(c2:Customer)
		WHERE id(c1) < id(c2)
		WITH c1, c2,
		     collect(DISTINCT {
		         type: type(r1),
		         identifier: CASE
		             %s
		             ELSE 'Unknown'
		         END
		     }) as sharedAttributes
		WHERE size(sharedAttributes) >= $minSharedAttributes
		WITH c1, c2, sharedAttributes, size(sharedAttributes) as sharedAttributeCount
		ORDER BY sharedAttributeCount DESC
		LIMIT $limit
		RETURN c1.customerId as customer1Id,
		       c1.firstName as customer1FirstName,
		       c1.lastName as customer1LastName,
		       c2.customerId as customer2Id,
		       c2.firstName as customer2FirstName,
		       c2.lastName as customer2LastName,
		       sharedAttributes,
		       sharedAttributeCount
	`, relPattern, relPattern, caseStatement)

	return query
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
