package schema

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mkd-neo4j/neo4j-mcp-fraud/internal/tools"
	"github.com/mkd-neo4j/neo4j-mcp-fraud/internal/tools/cypher"
)

const (
	defaultReferenceModelPath = "docs/fraud-mcp/DATA_MODEL.md"
	httpTimeout               = 30 * time.Second
)

var (
	// Default Neo4j reference model URLs
	defaultReferenceModelURLs = []string{
		"https://neo4j.com/developer/industry-use-cases/_attachments/transaction-base-model.txt",
		"https://neo4j.com/developer/industry-use-cases/_attachments/fraud-event-sequence-model.txt",
	}
)

// EnrichSchemaInput represents the input arguments for enrich-schema tool
type EnrichSchemaInput struct {
	ReferenceModelURLs string `json:"reference_model_urls,omitempty"`
	ReferenceModelPath string `json:"reference_model_path,omitempty"`
}

// EnrichSchemaHandler returns a handler function for the enrich-schema tool
func EnrichSchemaHandler(deps *tools.ToolDependencies, schemaSampleSize int32) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleEnrichSchema(ctx, deps, schemaSampleSize, request)
	}
}

// handleEnrichSchema enriches the raw schema with contextual information using LLM
func handleEnrichSchema(ctx context.Context, deps *tools.ToolDependencies, schemaSampleSize int32, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if deps.DBService == nil {
		errMessage := "database service is not initialized"
		slog.Error(errMessage)
		return mcp.NewToolResultError(errMessage), nil
	}

	if deps.AnalyticsService == nil {
		errMessage := "analytics service is not initialized"
		slog.Error(errMessage)
		return mcp.NewToolResultError(errMessage), nil
	}

	deps.AnalyticsService.EmitEvent(deps.AnalyticsService.NewToolsEvent("enrich-schema"))
	slog.Info("enriching schema with contextual information")

	// Step 1: Get raw schema from database
	rawSchemaResult, err := cypher.GetSchemaHandler(deps, schemaSampleSize)(ctx, mcp.CallToolRequest{})
	if err != nil {
		slog.Error("failed to retrieve raw schema", "error", err)
		return mcp.NewToolResultError(fmt.Sprintf("failed to retrieve raw schema: %v", err)), nil
	}

	if rawSchemaResult.IsError {
		return rawSchemaResult, nil
	}

	// Extract raw schema text from result
	var rawSchemaText string
	if len(rawSchemaResult.Content) > 0 {
		if textContent, ok := rawSchemaResult.Content[0].(mcp.TextContent); ok {
			rawSchemaText = textContent.Text
		} else {
			return mcp.NewToolResultError("unexpected schema result format"), nil
		}
	} else {
		return mcp.NewToolResultError("empty schema result"), nil
	}

	// Step 2: Load reference data models
	var args EnrichSchemaInput
	if err := request.BindArguments(&args); err != nil {
		slog.Warn("failed to bind arguments, using defaults", "error", err)
	}

	var referenceModels []string
	var referenceModelURLs []string

	// Parse URLs from args
	if args.ReferenceModelURLs != "" {
		referenceModelURLs = parseURLList(args.ReferenceModelURLs)
	}

	// If no parameters provided, use defaults
	if len(referenceModelURLs) == 0 && args.ReferenceModelPath == "" {
		referenceModelURLs = defaultReferenceModelURLs
	}

	// Fetch models from URLs
	if len(referenceModelURLs) > 0 {
		for _, url := range referenceModelURLs {
			content, err := fetchReferenceModelFromURL(ctx, url)
			if err != nil {
				slog.Warn("failed to fetch reference model from URL", "url", url, "error", err)
				continue
			}
			referenceModels = append(referenceModels, fmt.Sprintf("=== Reference Model from %s ===\n%s", url, content))
		}
	}

	// Load from local path if provided
	if args.ReferenceModelPath != "" {
		content, err := loadReferenceModelFromFile(args.ReferenceModelPath)
		if err != nil {
			slog.Warn("failed to load reference model from file", "path", args.ReferenceModelPath, "error", err)
		} else {
			referenceModels = append(referenceModels, fmt.Sprintf("=== Local Reference Model from %s ===\n%s", args.ReferenceModelPath, content))
		}
	}

	// Combine all reference models
	var combinedReferenceModel string
	if len(referenceModels) > 0 {
		combinedReferenceModel = strings.Join(referenceModels, "\n\n")
	} else {
		slog.Warn("no reference models could be loaded, proceeding without them")
		combinedReferenceModel = "No reference models available"
	}

	// Step 3: Create enrichment prompt for LLM
	enrichmentPrompt := buildEnrichmentPrompt(rawSchemaText, combinedReferenceModel)

	// Step 4: Return prompt as structured data for LLM client to process
	response := EnrichmentRequest{
		RawSchema:      rawSchemaText,
		ReferenceModel: combinedReferenceModel,
		Prompt:         enrichmentPrompt,
		Instructions: `This tool provides the raw database schema and reference data model for LLM-powered enrichment.

The LLM should:
1. Parse the raw schema to understand current database structure
2. Study the reference model to understand best practices and recommended patterns
3. Intelligently match nodes, relationships, and properties (handling fuzzy matches, synonyms, etc.)
4. Enrich each schema element with:
   - Business descriptions and meanings
   - Relationship semantics
   - Fraud detection context (if applicable)
   - Confidence scores for matches
   - Suggestions for missing recommended fields
   - Deviations from best practices
5. Return a structured JSON with enriched schema

Example enriched output format:
{
  "enrichedSchema": [
    {
      "key": "Customer",
      "value": {
        "type": "node",
        "description": "Represents a bank customer with identity verification",
        "matchConfidence": 0.95,
        "properties": {
          "customerId": {
            "type": "STRING",
            "description": "Unique customer identifier",
            "matchedReference": "customerId from Customer node",
            "confidence": 1.0
          }
        },
        "relationships": {
          "HAS_ACCOUNT": {
            "direction": "out",
            "labels": ["Account"],
            "description": "Links customer to their financial accounts"
          }
        },
        "missingRecommendedProperties": [
          {
            "name": "riskScore",
            "type": "FLOAT",
            "description": "Current calculated risk score (0-10)",
            "reason": "Recommended for fraud detection"
          }
        ]
      }
    }
  ],
  "summary": {
    "totalNodes": 5,
    "matchedNodes": 4,
    "deviations": ["Customer missing isPEP property"],
    "suggestions": ["Add fraud-specific properties to Customer node"]
  }
}`,
	}

	jsonResponse, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		slog.Error("failed to serialize enrichment request", "error", err)
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(string(jsonResponse)), nil
}

// EnrichmentRequest represents the data returned by enrich-schema tool
type EnrichmentRequest struct {
	RawSchema      string `json:"raw_schema"`
	ReferenceModel string `json:"reference_model"`
	Prompt         string `json:"prompt"`
	Instructions   string `json:"instructions"`
}

// parseURLList parses a comma-separated list of URLs
func parseURLList(urls string) []string {
	var result []string
	for _, url := range strings.Split(urls, ",") {
		url = strings.TrimSpace(url)
		if url != "" {
			result = append(result, url)
		}
	}
	return result
}

// fetchReferenceModelFromURL fetches a reference model from a URL
func fetchReferenceModelFromURL(ctx context.Context, url string) (string, error) {
	client := &http.Client{
		Timeout: httpTimeout,
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	return string(body), nil
}

// loadReferenceModelFromFile loads the reference data model from file
func loadReferenceModelFromFile(path string) (string, error) {
	// Try to resolve path relative to project root
	if !filepath.IsAbs(path) {
		// Try current working directory first
		if _, err := os.Stat(path); err == nil {
			content, err := os.ReadFile(path)
			if err != nil {
				return "", fmt.Errorf("failed to read reference model: %w", err)
			}
			return string(content), nil
		}

		// Try relative to executable
		execPath, err := os.Executable()
		if err == nil {
			absPath := filepath.Join(filepath.Dir(execPath), path)
			if _, err := os.Stat(absPath); err == nil {
				content, err := os.ReadFile(absPath)
				if err != nil {
					return "", fmt.Errorf("failed to read reference model: %w", err)
				}
				return string(content), nil
			}
		}

		return "", fmt.Errorf("reference model file not found: %s", path)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read reference model: %w", err)
	}
	return string(content), nil
}

// buildEnrichmentPrompt creates a comprehensive prompt for LLM enrichment
func buildEnrichmentPrompt(rawSchema, referenceModel string) string {
	return fmt.Sprintf(`You are a Neo4j data modeling expert specializing in graph database schemas and fraud detection patterns.

TASK:
Analyze the raw database schema and enrich it with contextual information by intelligently matching against Neo4j reference data models and best practices.

RAW DATABASE SCHEMA:
%s

REFERENCE DATA MODEL:
%s

INSTRUCTIONS:
1. Parse the raw schema to understand the current database structure (nodes, relationships, properties)
2. Study the reference model to understand recommended patterns, property descriptions, and best practices
3. Intelligently match schema elements:
   - Handle fuzzy matching (e.g., "cust_id" matches "customerId")
   - Recognize synonyms and variations in naming
   - Identify semantic similarities even with different names
   - Calculate confidence scores for each match (0.0 to 1.0)

4. For each matched node/relationship:
   - Add business descriptions
   - Enrich property meanings
   - Add relationship semantics
   - Include fraud detection context where relevant
   - Note deviations from best practices

5. Identify missing recommended elements:
   - Properties suggested by reference model but not in database
   - Relationships that should exist
   - Constraints or indexes that should be added

6. Return structured JSON with enriched schema including:
   - Descriptions for all elements
   - Match confidence scores
   - Missing recommended fields
   - Suggestions for improvements
   - Deviations from reference patterns

7. Be intelligent and flexible:
   - Don't require perfect matches
   - Use context to infer relationships
   - Handle industry-specific vs generic schemas
   - Provide value even with partial matches

OUTPUT FORMAT:
Return a JSON object with enriched schema and summary of findings.`, rawSchema, referenceModel)
}
