# Output Format Support

## Overview

Each fraud detection tool supports multiple output formats to enable different workflows within the Neo4j ecosystem. The key difference between formats is **variable templating** - the same query logic needs different parameter syntax depending on the execution context.

## Supported Formats

### 1. `json` (Default)
**Use case:** Direct investigation in Claude Desktop, MCP clients, or programmatic access

**Behavior:**
- Executes the Cypher query with provided parameters
- Returns result records as JSON
- Parameters are bound at execution time

**Example output:**
```json
{
  "results": [
    {
      "customer1": {"customerId": "C123", "firstName": "John"},
      "customer2": {"customerId": "C456", "firstName": "Jane"},
      "sharedDevice": {"deviceId": "D789"},
      "riskScore": 8.5
    }
  ],
  "summary": {
    "recordCount": 15,
    "executionTime": "245ms"
  }
}
```

### 2. `cypher`
**Use case:** Copy query to Neo4j Browser, Cypher Shell, or custom applications

**Behavior:**
- Returns Cypher query text with parameterized syntax
- Uses `$paramName` syntax for Neo4j drivers
- Parameters listed separately for binding

**Example output:**
```cypher
// Shared Devices - Identity Fraud Detection
// Parameters: $accountId, $minSharedAccounts

MATCH (target:Account {accountId: $accountId})
MATCH (target)-[:ACCESSED_BY]->(device:Device)<-[:ACCESSED_BY]-(other:Account)
WHERE other <> target
WITH target, device, collect(DISTINCT other) AS sharedAccounts
WHERE size(sharedAccounts) >= $minSharedAccounts
RETURN target, device, sharedAccounts, size(sharedAccounts) AS shareCount
ORDER BY shareCount DESC

// Parameters:
// - accountId: "A12345"
// - minSharedAccounts: 3
```

### 3. `bloom_saved_search`
**Use case:** Save query as Bloom Saved Search for analysts to run interactively

**Behavior:**
- Returns Cypher query with **Bloom variable syntax**: `{paramName}`
- Variables become input fields in Bloom UI
- Query optimized for Bloom's perspective rendering

**Example output:**
```cypher
// Shared Devices - Identity Fraud Detection
// Bloom Saved Search

MATCH (target:Account {accountId: {accountId}})
MATCH (target)-[:ACCESSED_BY]->(device:Device)<-[:ACCESSED_BY]-(other:Account)
WHERE other <> target
WITH target, device, collect(DISTINCT other) AS sharedAccounts
WHERE size(sharedAccounts) >= {minSharedAccounts}
RETURN target, device, sharedAccounts, size(sharedAccounts) AS shareCount
ORDER BY shareCount DESC
```

**Bloom variable types:**
- `{accountId}` - Text input field
- `{minSharedAccounts}` - Number input field

Analysts can then:
1. Open Bloom
2. Create new Saved Search
3. Paste this Cypher
4. Bloom automatically detects `{variables}` and creates input fields
5. Save with descriptive name: "Find Accounts Sharing Devices"

### 4. `bloom_scene_action`
**Use case:** Create context-menu action in Bloom scenes for right-click investigation workflows

**Behavior:**
- Returns Cypher query with **Scene Action variable syntax**: `$node` or `$relationship`
- Uses Bloom's built-in context variables
- Designed for "Expand from this node" patterns

**Example output:**
```cypher
// Shared Devices - Scene Action
// Right-click on Account node → Run Action → "Find Shared Devices"

MATCH (target:Account)
WHERE elementId(target) = $node
MATCH (target)-[:ACCESSED_BY]->(device:Device)<-[:ACCESSED_BY]-(other:Account)
WHERE other <> target
WITH target, device, collect(DISTINCT other) AS sharedAccounts
WHERE size(sharedAccounts) >= 3
RETURN target, device, sharedAccounts, size(sharedAccounts) AS shareCount
ORDER BY shareCount DESC
```

**Bloom Scene Action variables:**
- `$node` - The node that was right-clicked (auto-provided by Bloom)
- `$relationship` - The relationship that was right-clicked (for relationship actions)

Analysts can then:
1. Open Bloom scene with Account nodes
2. Right-click an Account
3. Actions → Create New Action
4. Paste this Cypher
5. Save as "Find Shared Devices"
6. Action appears in right-click menu for all Account nodes

## Parameter Mapping

### Tool Parameter → Format Syntax

| Tool Parameter | `json` | `cypher` | `bloom_saved_search` | `bloom_scene_action` |
|----------------|--------|----------|---------------------|---------------------|
| `accountId` | Bound at execution | `$accountId` | `{accountId}` | From `$node` |
| `minSharedAccounts` | Bound at execution | `$minSharedAccounts` | `{minSharedAccounts}` | Hardcoded or `{minSharedAccounts}` |
| `minTransactions` | Bound at execution | `$minTransactions` | `{minTransactions}` | Hardcoded or `{minTransactions}` |
| `maxHops` | Bound at execution | `$maxHops` | `{maxHops}` | Hardcoded or `{maxHops}` |

### Scene Action Design Pattern

Scene actions typically:
1. **Use `$node` or `$relationship`** to reference the clicked element
2. **Hardcode thresholds** (since there's no input UI) OR use `{variables}` for configurable actions
3. **Focus on expansion** - "from this node, show me..."
4. **Return visualization-friendly results** - nodes and relationships for graph rendering

## Implementation Schema

### Tool Input Schema

Each tool spec includes:

```go
{
    "type": "object",
    "properties": {
        "accountId": {
            "type": "string",
            "description": "Account ID to investigate"
        },
        "minSharedAccounts": {
            "type": "integer",
            "default": 3
        },
        "outputFormat": {
            "type": "string",
            "enum": ["json", "cypher", "bloom_saved_search", "bloom_scene_action"],
            "default": "json",
            "description": "Output format: json (execute and return results), cypher (parameterized query), bloom_saved_search (saved search with {vars}), bloom_scene_action (scene action with $node)"
        }
    },
    "required": ["accountId"]
}
```

### Handler Logic

```go
func Handler(deps *tools.ToolDependencies) mcp.ToolHandler {
    return func(args map[string]interface{}) (*mcp.CallToolResult, error) {
        // Parse parameters
        accountId := args["accountId"].(string)
        minShared := getInt(args, "minSharedAccounts", 3)
        outputFormat := getString(args, "outputFormat", "json")

        // Build base query
        query := buildSharedDevicesQuery()
        params := map[string]interface{}{
            "accountId": accountId,
            "minSharedAccounts": minShared,
        }

        // Handle output format
        switch outputFormat {
        case "json":
            return executeAndReturnJSON(deps, query, params)

        case "cypher":
            return formatAsCypherQuery(query, params)

        case "bloom_saved_search":
            return formatAsBloomSavedSearch(query, params)

        case "bloom_scene_action":
            return formatAsBloomSceneAction(query, params)

        default:
            return nil, fmt.Errorf("unsupported output format: %s", outputFormat)
        }
    }
}
```

### Format Functions

```go
// formatAsCypherQuery replaces params with $paramName syntax
func formatAsCypherQuery(query string, params map[string]interface{}) (*mcp.CallToolResult, error) {
    // Replace parameter placeholders with $syntax
    cypherQuery := query // Already uses $param syntax in base query

    // Build parameter documentation
    paramDocs := "// Parameters:\n"
    for key, value := range params {
        paramDocs += fmt.Sprintf("// - %s: %v\n", key, value)
    }

    return &mcp.CallToolResult{
        Content: []interface{}{
            mcp.TextContent{
                Type: "text",
                Text: cypherQuery + "\n\n" + paramDocs,
            },
        },
    }, nil
}

// formatAsBloomSavedSearch replaces $params with {params}
func formatAsBloomSavedSearch(query string, params map[string]interface{}) (*mcp.CallToolResult, error) {
    // Replace $paramName with {paramName}
    bloomQuery := strings.ReplaceAll(query, "$", "{")
    bloomQuery = strings.ReplaceAll(bloomQuery, "}", "}") // Fix any double replacements

    // Add Bloom-specific comments
    header := "// Bloom Saved Search\n// Variables: " + strings.Join(getParamNames(params), ", ") + "\n\n"

    return &mcp.CallToolResult{
        Content: []interface{}{
            mcp.TextContent{
                Type: "text",
                Text: header + bloomQuery,
            },
        },
    }, nil
}

// formatAsBloomSceneAction adapts query for scene action context
func formatAsBloomSceneAction(query string, params map[string]interface{}) (*mcp.CallToolResult, error) {
    // Replace account lookup with $node reference
    sceneQuery := adaptQueryForSceneAction(query, params)

    header := "// Bloom Scene Action\n// Right-click on Account node → Run Action\n\n"

    return &mcp.CallToolResult{
        Content: []interface{}{
            mcp.TextContent{
                Type: "text",
                Text: header + sceneQuery,
            },
        },
    }, nil
}
```

## Tool-Specific Examples

### Tool 1: `shared_devices`

#### As JSON (default)
```json
{
  "results": [
    {"customer1": {...}, "customer2": {...}, "device": {...}, "riskScore": 8.5}
  ]
}
```

#### As Cypher
```cypher
MATCH (target:Account {accountId: $accountId})
MATCH (target)-[:ACCESSED_BY]->(device:Device)<-[:ACCESSED_BY]-(other:Account)
WHERE other <> target
WITH target, device, collect(DISTINCT other) AS sharedAccounts
WHERE size(sharedAccounts) >= $minSharedAccounts
RETURN target, device, sharedAccounts

// Parameters:
// - accountId: "A12345"
// - minSharedAccounts: 3
```

#### As Bloom Saved Search
```cypher
MATCH (target:Account {accountId: {accountId}})
MATCH (target)-[:ACCESSED_BY]->(device:Device)<-[:ACCESSED_BY]-(other:Account)
WHERE other <> target
WITH target, device, collect(DISTINCT other) AS sharedAccounts
WHERE size(sharedAccounts) >= {minSharedAccounts}
RETURN target, device, sharedAccounts
```

#### As Bloom Scene Action
```cypher
MATCH (target:Account)
WHERE elementId(target) = $node
MATCH (target)-[:ACCESSED_BY]->(device:Device)<-[:ACCESSED_BY]-(other:Account)
WHERE other <> target
WITH target, device, collect(DISTINCT other) AS sharedAccounts
WHERE size(sharedAccounts) >= 3
RETURN target, device, sharedAccounts
```

## Bloom Documentation References

- **Saved Searches:** Use `{variableName}` syntax
- **Scene Actions:** Use `$node` or `$relationship` for context
- **Variable Detection:** Bloom automatically creates input fields for `{variables}`
- **Context Menu:** Scene actions appear in right-click menu on matching node/relationship types

## Testing Strategy

1. **Unit tests:** Verify format conversion functions
2. **Bloom integration tests:** Import queries into Bloom, verify variable detection
3. **Scene action tests:** Create scene actions, verify `$node` binding works
4. **Documentation:** Include screenshots of Bloom UI showing saved searches and scene actions

## Benefits

1. **Analyst Workflow Integration:** Queries can be saved and reused in Bloom
2. **Knowledge Sharing:** Teams can share saved searches and scene actions
3. **Progressive Investigation:** Start with Claude, save useful queries to Bloom for future use
4. **Flexible Consumption:** Same query logic, multiple execution contexts
5. **Bloom Catalog:** Build library of fraud investigation patterns in Bloom
