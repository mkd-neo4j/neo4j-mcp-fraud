package cypher

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mkd-neo4j/neo4j-mcp-fraud/internal/tools"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j/dbtype"
)

const (
	// schemaVisualizationQuery retrieves the graph structure (nodes and relationships)
	schemaVisualizationQuery = `CALL db.schema.visualization()`

	// nodePropertiesQuery retrieves node properties with their types
	nodePropertiesQuery = `
		CALL db.schema.nodeTypeProperties()
		YIELD nodeLabels, propertyName, propertyTypes
		RETURN nodeLabels, propertyName, propertyTypes
	`

	// relPropertiesQuery retrieves relationship properties with their types
	relPropertiesQuery = `
		CALL db.schema.relTypeProperties()
		YIELD relType, propertyName, propertyTypes
		RETURN relType, propertyName, propertyTypes
	`
)

// GetSchemaHandler returns a handler function for the get_schema tool
func GetSchemaHandler(deps *tools.ToolDependencies, schemaSampleSize int32) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleGetSchema(ctx, deps, schemaSampleSize)
	}
}

// handleGetSchema retrieves Neo4j schema information using native procedures
func handleGetSchema(ctx context.Context, deps *tools.ToolDependencies, schemaSampleSize int32) (*mcp.CallToolResult, error) {
	if deps.DBService == nil {
		errMessage := "database service is not initialized"
		slog.Error(errMessage)
		return mcp.NewToolResultError(errMessage), nil
	}
	// Emit analytics event
	if deps.AnalyticsService == nil {
		errMessage := "analytics service is not initialized"
		slog.Error(errMessage)
		return mcp.NewToolResultError(errMessage), nil
	}

	deps.AnalyticsService.EmitEvent(deps.AnalyticsService.NewToolsEvent("get-schema"))
	slog.Info("retrieving schema from the database", "database", deps.DBService.GetDatabaseName())

	// Execute schema visualization query to get graph structure
	visualizationRecords, err := deps.DBService.ExecuteReadQuery(ctx, schemaVisualizationQuery, nil)
	if err != nil {
		slog.Error("failed to execute schema visualization query", "error", err)
		return mcp.NewToolResultError(err.Error()), nil
	}

	slog.Debug("schema visualization query completed", "records_count", len(visualizationRecords))

	if len(visualizationRecords) == 0 {
		// Before declaring database empty, verify with a node count check
		slog.Warn("schema visualization returned no records, verifying database contents")
		countRecords, countErr := deps.DBService.ExecuteReadQuery(ctx, "MATCH (n) RETURN count(n) as nodeCount", nil)
		if countErr != nil {
			slog.Error("failed to execute node count verification query", "error", countErr)
			return mcp.NewToolResultError(fmt.Sprintf("schema visualization returned no records and verification failed: %v", countErr)), nil
		}

		if len(countRecords) > 0 {
			if nodeCount, ok := countRecords[0].Get("nodeCount"); ok {
				if count, ok := nodeCount.(int64); ok && count > 0 {
					slog.Error("database contains nodes but schema visualization returned empty",
						"nodeCount", count,
						"database", deps.DBService.GetDatabaseName())
					return mcp.NewToolResultError(fmt.Sprintf("Internal error: database '%s' contains %d nodes but schema visualization failed. This may indicate a schema introspection issue.", deps.DBService.GetDatabaseName(), count)), nil
				}
			}
		}

		slog.Info("database is empty, no schema to return", "database", deps.DBService.GetDatabaseName())
		return mcp.NewToolResultText(fmt.Sprintf("The get-schema tool executed successfully; however, since the Neo4j database '%s' contains no data, no schema information was returned.", deps.DBService.GetDatabaseName())), nil
	}

	// Execute node properties query
	nodePropsRecords, err := deps.DBService.ExecuteReadQuery(ctx, nodePropertiesQuery, nil)
	if err != nil {
		slog.Error("failed to execute node properties query", "error", err)
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Execute relationship properties query
	relPropsRecords, err := deps.DBService.ExecuteReadQuery(ctx, relPropertiesQuery, nil)
	if err != nil {
		slog.Error("failed to execute relationship properties query", "error", err)
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Process the three query results into unified schema
	structuredOutput, err := processNativeSchema(visualizationRecords, nodePropsRecords, relPropsRecords)
	if err != nil {
		slog.Error("failed to process get-schema native queries", "error", err)
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Convert to Neo4j documentation markdown format
	markdown := formatSchemaAsMarkdown(structuredOutput)

	// Add fraud detection context header
	const fraudDatabaseContext = `# Neo4j Fraud Detection Database Schema

This is a graph database for detecting and preventing financial crime. Graph databases excel at:
- **Pattern Detection**: Finding suspicious patterns across connected entities
- **Relationship Analysis**: Traversing networks to identify hidden connections
- **Identity Resolution**: Linking data points across multiple sources
- **Behavioral Analytics**: Detecting anomalies in transaction and activity patterns

**Example use cases** this type of database commonly supports include (but are not limited to):
- Detecting synthetic identities through shared PII analysis
- Identifying fraud rings and collusion networks
- Analyzing transaction flows for money laundering patterns
- Cross-referencing customer data for identity verification

The schema below shows the current structure of your Neo4j database.

---

`

	enrichedMarkdown := fraudDatabaseContext + markdown

	slog.Info("returning schema with fraud detection context", "schema_size", len(enrichedMarkdown))

	return mcp.NewToolResultText(enrichedMarkdown), nil
}

type SchemaItem struct {
	Key   string       `json:"key"`
	Value SchemaDetail `json:"value"`
}

type SchemaDetail struct {
	Type          string                  `json:"type"`
	Properties    map[string]string       `json:"properties,omitempty"`
	Relationships map[string]Relationship `json:"relationships,omitempty"`
}

type Relationship struct {
	Direction  string            `json:"direction"`
	Labels     []string          `json:"labels"` // List of target node labels
	Properties map[string]string `json:"properties,omitempty"`
}

// processNativeSchema combines results from native Neo4j schema procedures into a unified schema format
func processNativeSchema(visualizationRecords, nodePropsRecords, relPropsRecords []*neo4j.Record) ([]SchemaItem, error) {
	// Extract visualization data
	if len(visualizationRecords) == 0 {
		return nil, fmt.Errorf("no visualization records returned")
	}

	visRecord := visualizationRecords[0]
	nodesRaw, ok := visRecord.Get("nodes")
	if !ok {
		return nil, fmt.Errorf("missing 'nodes' in visualization record")
	}
	relationshipsRaw, ok := visRecord.Get("relationships")
	if !ok {
		return nil, fmt.Errorf("missing 'relationships' in visualization record")
	}

	// Parse nodes from visualization
	nodesList, ok := nodesRaw.([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid nodes format in visualization")
	}

	// Build node properties map: label -> {propName -> propType}
	nodePropMap := make(map[string]map[string]string)
	for _, record := range nodePropsRecords {
		nodeLabelsRaw, _ := record.Get("nodeLabels")
		propertyName, _ := record.Get("propertyName")
		propertyTypes, _ := record.Get("propertyTypes")

		if nodeLabels, ok := nodeLabelsRaw.([]interface{}); ok && len(nodeLabels) > 0 {
			if label, ok := nodeLabels[0].(string); ok {
				if propName, ok := propertyName.(string); ok {
					if propTypes, ok := propertyTypes.([]interface{}); ok && len(propTypes) > 0 {
						if propType, ok := propTypes[0].(string); ok {
							if nodePropMap[label] == nil {
								nodePropMap[label] = make(map[string]string)
							}
							nodePropMap[label][propName] = propType
						}
					}
				}
			}
		}
	}

	// Build relationship properties map: relType -> {propName -> propType}
	relPropMap := make(map[string]map[string]string)
	for _, record := range relPropsRecords {
		relTypeRaw, _ := record.Get("relType")
		propertyName, _ := record.Get("propertyName")
		propertyTypes, _ := record.Get("propertyTypes")

		if relType, ok := relTypeRaw.(string); ok {
			if propName, ok := propertyName.(string); ok {
				if propTypes, ok := propertyTypes.([]interface{}); ok && len(propTypes) > 0 {
					if propType, ok := propTypes[0].(string); ok {
						if relPropMap[relType] == nil {
							relPropMap[relType] = make(map[string]string)
						}
						relPropMap[relType][propName] = propType
					}
				}
			}
		}
	}

	// Parse relationships from visualization
	relationshipsList, ok := relationshipsRaw.([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid relationships format in visualization")
	}

	// Build node ID to label lookup map
	nodeIDToLabel := make(map[int64]string)
	for _, nodeRaw := range nodesList {
		// Try dbtype.Node first (real Neo4j driver)
		if node, ok := nodeRaw.(dbtype.Node); ok {
			if label, ok := node.Props["name"].(string); ok {
				nodeIDToLabel[node.Id] = label
				slog.Debug("mapped node ID to label", "id", node.Id, "label", label)
			} else {
				slog.Warn("skipping node: no name in Props", "props", node.Props)
			}
			continue
		}

		// Fallback to map for test mocks
		node, ok := nodeRaw.(map[string]interface{})
		if !ok {
			slog.Warn("skipping node: not dbtype.Node or map", "type", fmt.Sprintf("%T", nodeRaw))
			continue
		}
		// Get node ID - handle multiple numeric types
		var nodeID int64
		idRaw, exists := node["Id"]
		if !exists {
			slog.Warn("skipping node: no Id field", "node", node)
			continue
		}

		// Try different numeric types
		switch v := idRaw.(type) {
		case int64:
			nodeID = v
		case int:
			nodeID = int64(v)
		case int32:
			nodeID = int64(v)
		default:
			slog.Warn("skipping node: unsupported Id type", "type", fmt.Sprintf("%T", idRaw), "value", idRaw)
			continue
		}

		// Get node label from Props.name
		if props, ok := node["Props"].(map[string]interface{}); ok {
			if label, ok := props["name"].(string); ok {
				nodeIDToLabel[nodeID] = label
				slog.Debug("mapped node ID to label", "id", nodeID, "label", label)
			}
		} else {
			slog.Warn("skipping node: no Props field or wrong type", "node", node)
		}
	}

	slog.Info("built node ID to label map", "count", len(nodeIDToLabel))

	// Build node relationships map: nodeLabel -> {relType -> Relationship}
	nodeRelsMap := make(map[string]map[string]Relationship)
	for _, relRaw := range relationshipsList {
		// Try dbtype.Relationship first (real Neo4j driver)
		if rel, ok := relRaw.(dbtype.Relationship); ok {
			relType, ok := rel.Props["name"].(string)
			if !ok || relType == "" {
				slog.Warn("skipping relationship: no name in Props", "props", rel.Props)
				continue
			}

			startLabel := nodeIDToLabel[rel.StartId]
			endLabel := nodeIDToLabel[rel.EndId]

			if startLabel != "" && endLabel != "" {
				if nodeRelsMap[startLabel] == nil {
					nodeRelsMap[startLabel] = make(map[string]Relationship)
				}
				nodeRelsMap[startLabel][relType] = Relationship{
					Direction:  "out",
					Labels:     []string{endLabel},
					Properties: relPropMap[relType],
				}

				// Also add incoming relationship for end node
				if nodeRelsMap[endLabel] == nil {
					nodeRelsMap[endLabel] = make(map[string]Relationship)
				}
				nodeRelsMap[endLabel][relType] = Relationship{
					Direction:  "in",
					Labels:     []string{startLabel},
					Properties: relPropMap[relType],
				}
				slog.Debug("mapped relationship", "type", relType, "from", startLabel, "to", endLabel)
			}
			continue
		}

		// Fallback to map for test mocks
		rel, ok := relRaw.(map[string]interface{})
		if !ok {
			slog.Warn("skipping relationship: not dbtype.Relationship or map", "type", fmt.Sprintf("%T", relRaw))
			continue
		}

		// Extract StartId - handle multiple numeric types
		var startID int64
		startIDRaw, exists := rel["StartId"]
		if !exists {
			slog.Warn("skipping relationship: no StartId", "rel", rel)
			continue
		}
		switch v := startIDRaw.(type) {
		case int64:
			startID = v
		case int:
			startID = int64(v)
		case int32:
			startID = int64(v)
		default:
			slog.Warn("skipping relationship: unsupported StartId type", "type", fmt.Sprintf("%T", startIDRaw))
			continue
		}

		// Extract EndId - handle multiple numeric types
		var endID int64
		endIDRaw, exists := rel["EndId"]
		if !exists {
			slog.Warn("skipping relationship: no EndId", "rel", rel)
			continue
		}
		switch v := endIDRaw.(type) {
		case int64:
			endID = v
		case int:
			endID = int64(v)
		case int32:
			endID = int64(v)
		default:
			slog.Warn("skipping relationship: unsupported EndId type", "type", fmt.Sprintf("%T", endIDRaw))
			continue
		}

		// Get relationship type from Props.name
		relType := ""
		if props, ok := rel["Props"].(map[string]interface{}); ok {
			relType, _ = props["name"].(string)
		}
		if relType == "" {
			continue
		}

		// Look up node labels
		startLabel := nodeIDToLabel[startID]
		endLabel := nodeIDToLabel[endID]

		if startLabel != "" && endLabel != "" {
			if nodeRelsMap[startLabel] == nil {
				nodeRelsMap[startLabel] = make(map[string]Relationship)
			}
			nodeRelsMap[startLabel][relType] = Relationship{
				Direction:  "out",
				Labels:     []string{endLabel},
				Properties: relPropMap[relType],
			}

			// Also add incoming relationship for end node
			if nodeRelsMap[endLabel] == nil {
				nodeRelsMap[endLabel] = make(map[string]Relationship)
			}
			nodeRelsMap[endLabel][relType] = Relationship{
				Direction:  "in",
				Labels:     []string{startLabel},
				Properties: relPropMap[relType],
			}
		}
	}

	// Build final schema items
	result := make([]SchemaItem, 0)

	slog.Info("building final schema output", "nodeCount", len(nodesList), "relationshipCount", len(relationshipsList))

	// Add nodes
	for _, nodeRaw := range nodesList {
		var nodeName string

		// Try dbtype.Node first (real Neo4j driver)
		if node, ok := nodeRaw.(dbtype.Node); ok {
			if name, ok := node.Props["name"].(string); ok {
				nodeName = name
			}
		} else if node, ok := nodeRaw.(map[string]interface{}); ok {
			// Fallback to map for test mocks
			if props, ok := node["Props"].(map[string]interface{}); ok {
				nodeName, _ = props["name"].(string)
			}
		}

		if nodeName == "" {
			slog.Debug("skipping node in final output: no name")
			continue
		}

		result = append(result, SchemaItem{
			Key: nodeName,
			Value: SchemaDetail{
				Type:          "node",
				Properties:    nodePropMap[nodeName],
				Relationships: nodeRelsMap[nodeName],
			},
		})
		slog.Debug("added node to schema", "name", nodeName, "propCount", len(nodePropMap[nodeName]), "relCount", len(nodeRelsMap[nodeName]))
	}

	slog.Info("added nodes to schema", "count", len(result))

	// Add relationship types as separate items
	relTypesSeen := make(map[string]bool)
	for _, relRaw := range relationshipsList {
		var relType string

		// Try dbtype.Relationship first (real Neo4j driver)
		if rel, ok := relRaw.(dbtype.Relationship); ok {
			if name, ok := rel.Props["name"].(string); ok {
				relType = name
			}
		} else if rel, ok := relRaw.(map[string]interface{}); ok {
			// Fallback to map for test mocks
			if props, ok := rel["Props"].(map[string]interface{}); ok {
				relType, _ = props["name"].(string)
			}
		}

		if relType == "" || relTypesSeen[relType] {
			continue
		}
		relTypesSeen[relType] = true

		result = append(result, SchemaItem{
			Key: relType,
			Value: SchemaDetail{
				Type:       "relationship",
				Properties: relPropMap[relType],
			},
		})
	}

	slog.Info("schema processing complete", "totalItems", len(result), "nodes", len(result)-len(relTypesSeen), "relationshipTypes", len(relTypesSeen))
	return result, nil
}

// processCypherSchema is a func that transforms a list of Neo4j.Record in a JSON tagged struct,
// this allows us to maintain the same APOC query supported by multiple Neo4j versions while returning a tokens aware version of it.
// Properties are optimized to return directly the type and removing unnecessary information:
// From:
//
//	title: {
//	     unique: false,
//	     indexed: false,
//	     type: "STRING",
//	     existence: false
//	   }
//
// To:
// title: String
// Relationship,
// From:
//
//	 relationships:   {
//	    ALWAYS: {
//	      count: 16,
//	      direction: "out",
//	      labels: ["Something"],
//	      properties: {
//				releaseYear: {
//	      		unique: false,
//	      		indexed: false,
//	      		type: "STRING",
//	      		existence: false
//	    		}
//			 }
//	    }
//	  }
//
// To:
// { ALWAYS: { direction: "out", labels: ["ACTED_IN"], properties: { releaseYear: "DATE" } } }
// null values are stripped.
func processCypherSchema(records []*neo4j.Record) ([]SchemaItem, error) {
	simplifiedSchema := make([]SchemaItem, 0, len(records))

	for _, record := range records {
		// Extract "key" (e.g., "Movie", "ACTED_IN")
		keyRaw, ok := record.Get("key")
		if !ok {
			return nil, fmt.Errorf("missing 'key' column in record")
		}
		keyStr, ok := keyRaw.(string)
		if !ok {
			return nil, fmt.Errorf("invalid key returned")
		}

		// Extract "value" (The map containing properties, type, relationships)
		valRaw, ok := record.Get("value")
		if !ok {
			return nil, fmt.Errorf("missing 'value' column in record")
		}

		// Cast the value to a map
		data, ok := valRaw.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid value returned")
		}

		// Transformation logic.

		//  Extract Type ("node" or "relationship")
		itemType, ok := data["type"].(string)
		if !ok {
			return nil, fmt.Errorf("invalid type returned")
		}

		// Simplify Properties
		// Input:  { "name": { "type": "STRING", "indexed": ... } }
		// Output: { "name": "STRING" }
		cleanProps, ok := simplifyProperties(data["properties"])
		if !ok {
			return nil, fmt.Errorf("invalid properties returned")
		}

		// Simplify Relationships
		// Input:  { "CONNECTION": { "relationship": null, "direction": "out", "properties": {...} } }
		// Output: { "CONNECTION": { "direction": "out", "properties": {"dist": "FLOAT"} } }
		var cleanRels map[string]Relationship

		rawRels, relsExist := data["relationships"]
		// relationship can be nil
		if relsExist && rawRels != nil {
			if relsMap, ok := rawRels.(map[string]interface{}); ok && len(relsMap) > 0 {
				cleanRels = make(map[string]Relationship)
				for relName, rawRelDetails := range relsMap {
					relDetails, ok := rawRelDetails.(map[string]interface{})
					if !ok {
						return nil, fmt.Errorf("invalid relationship returned")
					}
					// Extract Direction
					direction, ok := relDetails["direction"].(string)
					if !ok {
						return nil, fmt.Errorf("invalid direction returned")
					}

					// Extract Target Labels
					var labels []string
					rawLabels, ok := relDetails["labels"].([]interface{})
					if !ok {
						return nil, fmt.Errorf("invalid relationship labels returned")
					}
					for _, l := range rawLabels {
						if lStr, ok := l.(string); ok {
							labels = append(labels, lStr)
						}
					}

					relProps, ok := simplifyProperties(relDetails["properties"])
					if !ok {
						return nil, fmt.Errorf("invalid relationship properties returned")
					}
					cleanRels[relName] = Relationship{
						Direction:  direction,
						Labels:     labels,
						Properties: relProps,
					}

				}
			}
		}

		simplifiedSchema = append(simplifiedSchema, SchemaItem{
			Key: keyStr,
			Value: SchemaDetail{
				Type:          itemType,
				Properties:    cleanProps,
				Relationships: cleanRels,
			},
		})
	}

	return simplifiedSchema, nil
}

// simplifyProperties removes all the not required information such as "existence", "indexed", "unique", and keep the type name.
func simplifyProperties(rawProps interface{}) (map[string]string, bool) {
	cleanProps := make(map[string]string)
	if props, ok := rawProps.(map[string]interface{}); ok {
		for propName, rawPropDetails := range props {
			if propDetails, ok := rawPropDetails.(map[string]interface{}); ok {
				if typeName, ok := propDetails["type"].(string); ok {
					cleanProps[propName] = typeName
				} else {
					return nil, false
				}
			}
		}
	} else {
		return nil, false
	}
	return cleanProps, true
}

// formatSchemaAsMarkdown converts the structured schema to Neo4j documentation markdown format
func formatSchemaAsMarkdown(items []SchemaItem) string {
	var md strings.Builder

	md.WriteString("# Database Schema\n\n")
	md.WriteString("This schema represents the current state of your Neo4j database.\n\n")

	// Separate nodes and relationships
	var nodes []SchemaItem
	var relationships []SchemaItem

	for _, item := range items {
		if item.Value.Type == "node" {
			nodes = append(nodes, item)
		} else if item.Value.Type == "relationship" {
			relationships = append(relationships, item)
		}
	}

	// Write nodes section
	if len(nodes) > 0 {
		md.WriteString("## 1. Node Labels and Properties\n\n")

		for _, node := range nodes {
			md.WriteString(fmt.Sprintf("### %s\n\n", node.Key))

			// Write properties
			if len(node.Value.Properties) > 0 {
				md.WriteString("*Properties:*\n\n")
				for propName, propType := range node.Value.Properties {
					md.WriteString(fmt.Sprintf("  - `%s` (%s)\n", propName, propType))
				}
				md.WriteString("\n")
			}

			// Write relationships
			if len(node.Value.Relationships) > 0 {
				md.WriteString("*Relationships:*\n\n")
				for relName, rel := range node.Value.Relationships {
					// Format as proper Cypher: (:Source)-[:REL_TYPE]->(:Target) or (:Source)<-[:REL_TYPE]-(:Target)
					var cypherPattern string
					targetLabels := strings.Join(rel.Labels, ", ")
					if rel.Direction == "out" {
						cypherPattern = fmt.Sprintf("(:%s)-[:%s]->(:%s)", node.Key, relName, targetLabels)
					} else {
						cypherPattern = fmt.Sprintf("(:%s)<-[:%s]-(:%s)", node.Key, relName, targetLabels)
					}
					md.WriteString(fmt.Sprintf("  - `%s`\n", cypherPattern))
				}
				md.WriteString("\n")
			}
		}
	}

	// Write relationships section
	if len(relationships) > 0 {
		md.WriteString("## 2. Relationship Types\n\n")

		for _, rel := range relationships {
			md.WriteString(fmt.Sprintf("### :%s\n\n", rel.Key))

			if len(rel.Value.Properties) > 0 {
				md.WriteString("*Properties:*\n\n")
				for propName, propType := range rel.Value.Properties {
					md.WriteString(fmt.Sprintf("  - `%s` (%s)\n", propName, propType))
				}
				md.WriteString("\n")
			}
		}
	}

	return md.String()
}
