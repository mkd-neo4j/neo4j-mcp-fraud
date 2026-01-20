//go:build integration

package integration

import (
	"testing"

	"github.com/mkd-neo4j/neo4j-mcp-fraud/internal/tools/cypher/read"
	"github.com/mkd-neo4j/neo4j-mcp-fraud/internal/tools/cypher/write"
	"github.com/mkd-neo4j/neo4j-mcp-fraud/test/integration/helpers"
)

func TestWriteThenRead(t *testing.T) {
	t.Parallel()
	tc := helpers.NewTestContext(t, dbs.GetDriver())

	companyLabel := tc.GetUniqueLabel("Company")

	writeHandler := write.WriteCypherHandler(tc.Deps)
	tc.CallTool(writeHandler, map[string]any{
		"query":  "CREATE (c:" + companyLabel + " {name: $name, industry: $industry}) RETURN c",
		"params": map[string]any{"name": "Neo4j", "industry": "Database"},
	})

	readHandler := read.ReadCypherHandler(tc.Deps)
	res := tc.CallTool(readHandler, map[string]any{
		"query":  "MATCH (c:" + companyLabel + ") RETURN c",
		"params": map[string]any{},
	})

	var records []map[string]any
	tc.ParseJSONResponse(res, &records)

	if len(records) != 1 {
		t.Fatalf("expected 1 company, got %d", len(records))
	}

	company := records[0]["c"].(map[string]any)
	tc.AssertNodeProperties(company, map[string]any{
		"name":     "Neo4j",
		"industry": "Database",
	})
	tc.AssertNodeHasLabel(company, companyLabel)
}
