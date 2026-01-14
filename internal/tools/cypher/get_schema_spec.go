package cypher

import (
	"github.com/mark3labs/mcp-go/mcp"
)

func GetSchemaSpec() mcp.Tool {
	return mcp.NewTool("get-schema",
		mcp.WithDescription(`
		Retrieve the raw structural schema from the Neo4j database, including node labels, relationship types, and property keys.
		Returns simplified schema with property types and relationship directions.

		If the database contains no data, no schema information is returned.

		WORKFLOW FOR ENRICHED SCHEMA:
		For comprehensive schema understanding with business context and best practices:
		1. Call get-schema to retrieve raw database structure
		2. Call enrich-schema to get LLM enrichment prompt with reference models
		3. Use both results together for intelligent schema analysis and query generation

		The raw schema provides structural information (what exists in the database).
		The enrich-schema tool adds semantic context (what it means and best practices).`),
		mcp.WithTitleAnnotation("Get Neo4j Schema"),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithOpenWorldHintAnnotation(true),
	)
}
