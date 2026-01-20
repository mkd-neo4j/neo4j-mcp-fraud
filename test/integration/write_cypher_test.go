//go:build integration

package integration

import (
	"testing"

	"github.com/mkd-neo4j/neo4j-mcp-fraud/internal/tools/cypher/write"
	"github.com/mkd-neo4j/neo4j-mcp-fraud/test/integration/helpers"
)

func TestWriteCypher(t *testing.T) {
	t.Parallel()
	tc := helpers.NewTestContext(t, dbs.GetDriver())

	personLabel := tc.GetUniqueLabel("Person")

	write := write.WriteCypherHandler(tc.Deps)
	tc.CallTool(write, map[string]any{
		"query":  "CREATE (p:" + personLabel + " {name: $name}) RETURN p",
		"params": map[string]any{"name": "Alice"},
	})

	tc.VerifyNodeInDB(personLabel, map[string]any{"name": "Alice"})
}
