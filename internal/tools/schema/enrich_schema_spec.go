package schema

import (
	"github.com/mark3labs/mcp-go/mcp"
)

func EnrichSchemaSpec() mcp.Tool {
	return mcp.NewTool("enrich-schema",
		mcp.WithDescription(`
		Provides enrichment context and LLM prompt for intelligent schema analysis.

		PREREQUISITE: Call get-schema first to retrieve the raw database schema.

		WORKFLOW:
		1. First, call get-schema to retrieve raw database structure
		2. Then, call enrich-schema to fetch Neo4j reference models and get enrichment prompt
		3. Use the LLM prompt and reference models to intelligently match and enrich the raw schema

		This tool automatically fetches Neo4j reference data models (transaction models, fraud detection patterns)
		and returns a structured prompt for LLM-powered enrichment that:

		- Adds property descriptions and business meanings
		- Enriches relationship semantics
		- Identifies alignment with Neo4j best practices
		- Suggests missing recommended properties
		- Flags deviations from reference patterns
		- Handles fuzzy matching for property and node names (e.g., "cust_id" → "customerId")

		The enrichment provides rich business context for:
		- Understanding the domain model
		- Generating better Cypher queries
		- Fraud detection and financial services use cases
		- Identifying missing security or compliance fields

		Optional parameters:
		- reference_model_urls: Comma-separated list of URLs to fetch reference data models from
		  (e.g., https://neo4j.com/developer/industry-use-cases/_attachments/transaction-base-model.txt)
		- reference_model_path: Path to local reference data model documentation file

		If neither is provided, defaults to Neo4j official fraud detection and transaction models.

		RETURNS: JSON with raw_schema (from get-schema), reference_model, prompt, and instructions for enrichment.
		`),
		mcp.WithString("reference_model_urls",
			mcp.Description("Comma-separated list of URLs to Neo4j reference data model files"),
		),
		mcp.WithString("reference_model_path",
			mcp.Description("Path to local reference data model documentation file"),
		),
		mcp.WithTitleAnnotation("Enrich Neo4j Schema with Context"),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithOpenWorldHintAnnotation(true),
	)
}
