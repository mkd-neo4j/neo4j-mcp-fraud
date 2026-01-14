package cypher

import (
	"github.com/mark3labs/mcp-go/mcp"
)

func GetSchemaSpec() mcp.Tool {
	return mcp.NewTool("get-schema",
		mcp.WithDescription(`
		Retrieve the database schema from Neo4j with fraud detection context.

		Returns the structure of your Neo4j database including:
		- Node labels and their properties with data types
		- Relationship types and their directions
		- Fraud detection context explaining the purpose of this database

		This tool provides complete schema information with business context in one call.

		If the database contains no data, no schema information is returned.`),
		mcp.WithTitleAnnotation("Get Neo4j Schema"),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithOpenWorldHintAnnotation(true),
	)
}
