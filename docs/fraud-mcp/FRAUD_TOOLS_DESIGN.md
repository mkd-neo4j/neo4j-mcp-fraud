# Fraud Detection Tools - Design Document

## Overview

This document specifies the design for a specialized set of fraud detection tools to be added to the Neo4j MCP server. These tools are purpose-built for financial crime analysts, AML investigators, and compliance officers conducting fraud investigations.

### Goals

1. **Accelerate fraud investigations** by providing pre-built, graph-optimized queries for common investigation patterns
2. **Lower the barrier to entry** for analysts who need to query Neo4j but may not be Cypher experts
3. **Leverage Neo4j's strengths** in pattern matching, deep traversals, and relationship analysis
4. **Build a foundation** for future agentic workflows and tool orchestration

### Target Users

- **Fraud Analysts**: Investigating suspicious transactions and customer behavior
- **AML Investigators**: Conducting anti-money laundering investigations
- **KYC/CDD Officers**: Performing know-your-customer and customer due diligence checks
- **Compliance Teams**: Monitoring for regulatory compliance

### Key Use Cases (from stakeholder conversation)

Based on real-world requirements from fraud investigation teams:

1. **Transaction Monitoring**: Detecting unusual transaction patterns, circular flows, velocity anomalies
2. **KYC/CDD Compliance**: Validating documentation completeness, identifying missing/expired documents
3. **Network Analysis**: Finding connections to known bad actors, PEPs (Politically Exposed Persons)
4. **Device & Identity Fraud**: Detecting shared devices, credentials, and addresses
5. **Risk Assessment**: Evaluating exposure to high-risk jurisdictions and entities

---


## Data Model

The fraud detection tools use the official Neo4j Transaction & Account Data Model with fraud-specific extensions.

**For complete data model details, node types, relationship types, and schema definitions, see: [DATA_MODEL.md](DATA_MODEL.md)**

Key highlights:
- Based on official Neo4j Transaction & Account Base Model
- Extends with Fraud Event Sequence Model for account takeover detection
- Includes proposed fraud-specific properties (riskScore, isPEP, isFraudster, etc.)
- Comprehensive constraints and indexes for optimal query performance

---

## Integration Architecture

### Package Structure

Starting with 3 core fraud detection tools that demonstrate key investigation patterns:

```
internal/tools/fraud/
├── README.md                              # Package overview and usage
├── register.go                            # Tool registration (exports all tools to server)
├── types.go                               # Shared types and interfaces
├── risk_scorer.go                         # Risk scoring utilities
│
├── shared_devices/                        # Tool 1: Device/credential sharing detection
│   ├── handler.go                         # Implementation
│   ├── handler_test.go                    # Unit tests
│   └── spec.go                            # MCP tool specification
│
├── circular_flows/                        # Tool 2: Circular payment detection
│   ├── handler.go                         # Implementation
│   ├── handler_test.go                    # Unit tests
│   └── spec.go                            # MCP tool specification
│
└── bad_actors/                            # Tool 3: Path to known bad actors
    ├── handler.go                         # Implementation
    ├── handler_test.go                    # Unit tests
    └── spec.go                            # MCP tool specification
```

**Why these 3 tools?**

1. **`shared_devices`** - Demonstrates **identity fraud** patterns (device sharing, fraud rings)
2. **`circular_flows`** - Demonstrates **transaction monitoring** patterns (money laundering, structuring)
3. **`bad_actors`** - Demonstrates **network analysis** patterns (graph traversal, risk propagation)

These cover the three main fraud investigation categories and can be easily extended later.

### Shared Utilities

**`internal/tools/fraud/types.go`**:
```go
package fraud

import "github.com/neo4j/mcp/internal/tools"

// Common risk levels
const (
    RiskLevelLow      = "low"
    RiskLevelMedium   = "medium"
    RiskLevelHigh     = "high"
    RiskLevelCritical = "critical"
)

// RiskAssessment contains risk scoring information
type RiskAssessment struct {
    RiskLevel   string   `json:"riskLevel"`
    RiskScore   float64  `json:"riskScore"`
    Confidence  float64  `json:"confidence"`
    Factors     []string `json:"factors"`
}

// ToolDeps is an alias for shared tool dependencies
type ToolDeps = tools.ToolDependencies
```

**`internal/tools/fraud/risk_scorer.go`**:
```go
package fraud

// CalculateRiskLevel determines risk level from numeric score
func CalculateRiskLevel(score float64) string {
    switch {
    case score >= 8.0:
        return RiskLevelCritical
    case score >= 6.0:
        return RiskLevelHigh
    case score >= 4.0:
        return RiskLevelMedium
    default:
        return RiskLevelLow
    }
}

// Additional risk scoring utilities...
```

### Tool Registration

**`internal/tools/fraud/register.go`**:

This file centralizes all fraud tool registrations, providing a clean API for the server:

```go
package fraud

import (
    "github.com/mark3labs/mcp-go/server"
    "github.com/neo4j/mcp/internal/tools"
    "github.com/neo4j/mcp/internal/tools/fraud/shared_devices"
    "github.com/neo4j/mcp/internal/tools/fraud/circular_flows"
    "github.com/neo4j/mcp/internal/tools/fraud/bad_actors"
)

// RegisterAll registers all fraud detection tools with the MCP server
func RegisterAll(mcpServer *server.MCPServer, deps *tools.ToolDependencies) {
    // Tool 1: Device & Identity Fraud
    mcpServer.AddTool(shared_devices.Spec(), shared_devices.Handler(deps))

    // Tool 2: Transaction Monitoring
    mcpServer.AddTool(circular_flows.Spec(), circular_flows.Handler(deps))

    // Tool 3: Network Analysis
    mcpServer.AddTool(bad_actors.Spec(), bad_actors.Handler(deps))
}
```

**Update `internal/server/tools_register.go`**:

```go
import (
    "github.com/neo4j/mcp/internal/tools/fraud"
    "github.com/neo4j/mcp/internal/tools/cypher"
    "github.com/neo4j/mcp/internal/tools/gds"
    // ... existing imports
)

func (s *Neo4jMCPServer) registerTools() error {
    deps := &tools.ToolDependencies{
        DBService:        s.dbService,
        AnalyticsService: s.anService,
    }

    // Core Neo4j tools
    s.MCPServer.AddTool(cypher.GetSchemaSpec(), cypher.GetSchemaHandler(deps))
    s.MCPServer.AddTool(cypher.ReadCypherSpec(), cypher.ReadCypherHandler(deps))

    if !s.config.ReadOnly {
        s.MCPServer.AddTool(cypher.WriteCypherSpec(), cypher.WriteCypherHandler(deps))
    }

    // Fraud detection tools (all read-only)
    fraud.RegisterAll(s.MCPServer, deps)

    // GDS tools (if available)
    if s.gdsInstalled {
        s.MCPServer.AddTool(gds.ListGDSProceduresSpec(), gds.ListGDSProceduresHandler(deps))
    }

    return nil
}
```

### Example Tool Implementation

**`internal/tools/fraud/shared_devices/spec.go`**:

```go
package shared_devices

import "github.com/mark3labs/mcp-go/mcp"

// Spec returns the MCP tool specification
func Spec() mcp.Tool {
    return mcp.Tool{
        Name: "find-shared-devices",
        Description: `Identifies customers sharing devices with a target customer.

**Use Cases**: Fraud ring detection, account takeover investigation
**Risk Indicators**: Device sharing suggests coordinated fraud
**Investigation Tips**: Look for clusters sharing multiple devices`,
        InputSchema: mcp.ToolInputSchema{
            Type: "object",
            Properties: map[string]interface{}{
                "customerId": map[string]interface{}{
                    "type":        "string",
                    "description": "Customer ID to investigate",
                },
                "deviceType": map[string]interface{}{
                    "type":        "string",
                    "description": "Device type filter",
                    "enum":        []string{"mobile", "desktop", "tablet", "all"},
                    "default":     "all",
                },
            },
            Required: []string{"customerId"},
        },
    }
}
```

**`internal/tools/fraud/shared_devices/handler.go`**:

```go
package shared_devices

import (
    "context"
    "log/slog"

    "github.com/mark3labs/mcp-go/mcp"
    "github.com/neo4j/mcp/internal/tools/fraud"
)

// Handler returns the tool handler function
func Handler(deps *fraud.ToolDeps) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
        // Emit analytics
        deps.AnalyticsService.EmitEvent(
            deps.AnalyticsService.NewToolsEvent("find-shared-devices"),
        )

        // Parse arguments
        var args struct {
            CustomerId string `json:"customerId"`
            DeviceType string `json:"deviceType"`
        }
        if err := request.BindArguments(&args); err != nil {
            slog.Error("error binding arguments", "error", err)
            return mcp.NewToolResultError(err.Error()), nil
        }

        // Execute query
        query := `
            MATCH (target:Customer {customerId: $customerId})<-[:USED_BY]-(device:Device)
            WHERE device.deviceType = $deviceType OR $deviceType = 'all'
            MATCH (device)-[:USED_BY]->(other:Customer)
            WHERE target.customerId <> other.customerId
            WITH other, collect(DISTINCT device) as sharedDevices
            RETURN other.customerId as customerId,
                   other.firstName + ' ' + other.lastName as customerName,
                   size(sharedDevices) as deviceCount
            ORDER BY deviceCount DESC
            LIMIT 100
        `

        records, err := deps.DBService.ExecuteReadQuery(ctx, query, map[string]any{
            "customerId": args.CustomerId,
            "deviceType": args.DeviceType,
        })
        if err != nil {
            slog.Error("query execution failed", "error", err)
            return mcp.NewToolResultError(err.Error()), nil
        }

        // Format response
        response, err := deps.DBService.Neo4jRecordsToJSON(records)
        if err != nil {
            slog.Error("result formatting failed", "error", err)
            return mcp.NewToolResultError(err.Error()), nil
        }

        return mcp.NewToolResultText(response), nil
    }
}
```

### Analytics Tracking

Add fraud-specific analytics events in `internal/analytics/events.go`:

```go
func (a *Analytics) NewFraudToolEvent(toolName string) Event {
    return Event{
        EventName: "fraud_tool_used",
        Properties: map[string]interface{}{
            "tool_name":    toolName,
            "category":     "fraud_detection",
            "mcp_version":  a.mcpVersion,
        },
    }
}
```

Update each fraud handler to emit analytics:

```go
deps.AnalyticsService.EmitEvent(
    deps.AnalyticsService.NewFraudToolEvent("find-shared-devices")
)
```

---

## Documentation & User Experience

### README Updates

Add fraud tools section to main README:

```markdown
## Fraud Detection Tools

This MCP server includes specialized tools for financial crime investigation:

### Available Tools

1. **`find-shared-devices`** - Detect device/credential sharing patterns
   - Identifies fraud rings and account takeover attempts
   - Finds customers sharing IP addresses, device IDs, or other digital fingerprints

2. **`detect-circular-flows`** - Identify circular payment patterns
   - Detects potential money laundering through circular fund movements
   - Traces money returning to source through intermediary accounts

3. **`path-to-bad-actors`** - Find connections to known fraudsters/PEPs
   - Discovers shortest paths to flagged entities
   - Assesses risk through network proximity to bad actors

### Example Prompts

Use natural language with Claude Desktop:

- "Find all customers sharing devices with customer CUS123"
- "Check if customer CUS456 has any circular payment flows"
- "Show me anyone connected to known fraudsters within 4 hops"
- "Investigate the network around this suspicious account"
```

### Tool Descriptions (Best Practices for LLM Selection)

Each tool spec must have **rich, semantic descriptions** that maximize LLM comprehension and correct tool selection.

**Critical elements for effective tool descriptions:**

1. **WHEN to use** - Investigation scenarios that trigger this tool
2. **WHAT it detects** - Specific fraud patterns and behaviors
3. **WHY it matters** - Risk indicators and severity levels
4. **HOW to interpret** - Investigation workflow and follow-up actions
5. **EXAMPLES** - Natural language queries that map to this tool

**Example Pattern:**

```go
func Spec() mcp.Tool {
    return mcp.Tool{
        Name: "find-shared-devices",  // Action-oriented, clear purpose
        Description: `[CONCISE SUMMARY] + [DETAILED CONTEXT]

**When to use this tool:**
- [Scenario 1: User intent trigger]
- [Scenario 2: Investigation context]

**What it detects:**
- [Pattern 1 with specific behavior]
- [Pattern 2 with fraud indicator]

**Fraud indicators this reveals:**
- CRITICAL: [High-severity pattern]
- HIGH RISK: [Moderate-severity pattern]
- MEDIUM RISK: [Lower-severity pattern]

**Investigation workflow:**
1. [Step-by-step guidance]
2. [Follow-up actions]

**Example scenarios:**
- "Check if customer X shares devices with others"
- "Find fraud rings around account Y"`,
        InputSchema: // Detailed parameter descriptions
    }
}
```

**Why this works:**
- ✅ **Semantic matching**: LLM maps user intent to tool purpose
- ✅ **Context awareness**: LLM understands WHEN to use each tool
- ✅ **Natural language**: Example queries help LLM pattern match
- ✅ **Risk framing**: LLM can explain results appropriately
- ✅ **Workflow guidance**: LLM suggests logical next steps

**Tool naming conventions:**
- Use action verbs: `find-`, `detect-`, `analyze-`, `check-`
- Be specific: `shared-devices` not `device-query`
- Match user language: `bad-actors` not `fraudster-graph`

---

