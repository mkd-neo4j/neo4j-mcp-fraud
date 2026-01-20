package read

import (
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mkd-neo4j/neo4j-mcp-fraud/internal/tools/cypher/utils"
)

type ReadCypherInput struct {
	Query  string       `json:"query" jsonschema:"default=MATCH(n) RETURN n,description=The Cypher query to execute"`
	Params utils.Params `json:"params,omitempty" jsonschema:"default={},description=Parameters to pass to the Cypher query"`
}

func ReadCypherSpec() mcp.Tool {
	return mcp.NewTool("read-cypher",
		mcp.WithDescription("read-cypher can run only read-only Cypher statements. For write operations (CREATE, MERGE, DELETE, SET, etc...), schema/admin commands, or PROFILE queries, use write-cypher instead."),
		mcp.WithInputSchema[ReadCypherInput](),
		mcp.WithTitleAnnotation("Read Cypher"),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(true),
	)
}
