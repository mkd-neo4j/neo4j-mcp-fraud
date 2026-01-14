package tools

import (
	"github.com/mkd-neo4j/neo4j-mcp-fraud/internal/analytics"
	"github.com/mkd-neo4j/neo4j-mcp-fraud/internal/database"
)

// ToolDependencies contains all dependencies needed by tools
type ToolDependencies struct {
	DBService        database.Service
	AnalyticsService analytics.Service
	SchemaSampleSize int
}
