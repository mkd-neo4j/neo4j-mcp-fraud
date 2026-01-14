# Analytics & Query Harvesting

## Overview

A dedicated Neo4j database captures every fraud detection query, user question, and tool invocation to enable:

1. **Pattern Recognition** - Identify most common investigation workflows
2. **Tool Optimization** - Understand which tools are most valuable
3. **Query Catalog Building** - Harvest successful queries for future reuse
4. **User Behavior Analysis** - Learn how analysts interact with fraud data
5. **Tool Evolution** - Data-driven decisions on new tools to build

## High-Level Architecture

```
┌─────────────────────┐
│   Claude Desktop    │
│   (or MCP Client)   │
└──────────┬──────────┘
           │
           │ 1. User asks question
           ▼
┌─────────────────────┐
│   Neo4j MCP Server  │
│                     │
│  ┌───────────────┐  │
│  │ Fraud Tools   │  │
│  └───────┬───────┘  │
│          │          │
│          │ 2. Log event
│          ▼          │
│  ┌───────────────┐  │
│  │ Analytics     │  │
│  │ Service       │  │
│  └───────┬───────┘  │
└──────────┼──────────┘
           │
           │ 3. Write to analytics DB
           ▼
┌─────────────────────┐         ┌─────────────────────┐
│  Analytics Neo4j DB │         │  Fraud Data Neo4j   │
│  (Separate)         │         │  (Investigation DB) │
│                     │         │                     │
│  - User questions   │         │  - Customers        │
│  - Tool invocations │         │  - Accounts         │
│  - Query patterns   │         │  - Transactions     │
│  - Timing metrics   │         │  - Devices          │
│  - Output formats   │         │  - Fraud rings      │
└─────────────────────┘         └─────────────────────┘
```

## Core Concepts

### 1. Separate Analytics Database

**Why separate?**
- **Isolation** - Analytics don't pollute production fraud data
- **Performance** - Heavy analytical queries don't impact investigations
- **Security** - Different access control (analytics may be more permissive)
- **Lifecycle** - Analytics can be archived/purged independently
- **Schema freedom** - Analytics model can evolve without affecting fraud schema

**Connection configuration:**
```env
# Main fraud investigation database
NEO4J_URI=bolt://localhost:7687
NEO4J_DATABASE=fraud

# Analytics/harvesting database (separate instance or same instance, different DB)
ANALYTICS_NEO4J_URI=bolt://localhost:7687
ANALYTICS_NEO4J_DATABASE=fraud_analytics
```

### 2. Data Capture Points

**Every fraud tool invocation captures:**

1. **User Question** (original natural language)
2. **Tool Selected** (which tool was chosen)
3. **Parameters Used** (what values were provided)
4. **Output Format** (json, cypher, bloom_saved_search, bloom_scene_action)
5. **Query Executed** (the actual Cypher that ran)
6. **Execution Metrics** (timing, record count, success/failure)
7. **Context** (user ID, session ID, timestamp)
8. **Results Metadata** (not full results, but summary - e.g., "found 5 fraud rings")

### 3. Graph Model for Analytics

**Conceptual schema:**

```
(:Session)-[:ASKED]->(:Question)
(:Question)-[:MAPPED_TO]->(:ToolInvocation)
(:ToolInvocation)-[:USED_TOOL]->(:Tool)
(:ToolInvocation)-[:EXECUTED]->(:CypherQuery)
(:ToolInvocation)-[:RETURNED]->(:ResultSummary)
(:Question)-[:SIMILAR_TO]->(:Question)  // Pattern clustering
(:Tool)-[:FREQUENTLY_FOLLOWED_BY]->(:Tool)  // Workflow patterns
```

**Node types:**

- **Session** - A Claude Desktop conversation or MCP client session
- **Question** - Natural language question from user
- **ToolInvocation** - Single tool execution event
- **Tool** - Fraud tool definition (shared_devices, circular_flows, bad_actors)
- **CypherQuery** - The actual query that ran (with parameters)
- **ResultSummary** - Metadata about results (count, timing, risk scores found)
- **User** - User identity (if available)
- **OutputFormat** - json, cypher, bloom_saved_search, bloom_scene_action

**Relationships:**

- `(:Session)-[:ASKED]->(:Question)` - Questions asked in a session
- `(:Question)-[:MAPPED_TO]->(:ToolInvocation)` - LLM selected this tool for this question
- `(:ToolInvocation)-[:USED_TOOL]->(:Tool)` - Which tool was invoked
- `(:ToolInvocation)-[:EXECUTED]->(:CypherQuery)` - Query that was run
- `(:ToolInvocation)-[:WITH_PARAMETERS]->(:ParameterSet)` - Parameters used
- `(:ToolInvocation)-[:RETURNED]->(:ResultSummary)` - What came back
- `(:ToolInvocation)-[:OUTPUT_FORMAT]->(:OutputFormat)` - How results were formatted
- `(:Question)-[:SIMILAR_TO]->(:Question)` - Semantic similarity (for clustering)
- `(:Tool)-[:FOLLOWED_BY]->(:Tool)` - Sequential tool usage patterns
- `(:User)-[:INITIATED]->(:Session)` - User sessions

### 4. What Gets Harvested

#### A. User Questions
```cypher
CREATE (q:Question {
  questionId: "q-uuid",
  text: "Show me all accounts that share devices with known fraudsters",
  askedAt: datetime(),
  tokenCount: 15,
  language: "en"
})
```

#### B. Tool Selection
```cypher
CREATE (ti:ToolInvocation {
  invocationId: "ti-uuid",
  toolName: "shared_devices",
  selectedAt: datetime(),
  selectionConfidence: 0.95  // If LLM provides confidence
})

CREATE (q)-[:MAPPED_TO]->(ti)
```

#### C. Query Execution
```cypher
CREATE (cq:CypherQuery {
  queryId: "cq-uuid",
  queryText: "MATCH (target:Account {accountId: $accountId})...",
  queryHash: "sha256-hash",  // Deduplicate identical queries
  executedAt: datetime()
})

CREATE (ps:ParameterSet {
  parameters: {
    accountId: "A12345",
    minSharedAccounts: 3,
    outputFormat: "json"
  }
})

CREATE (ti)-[:EXECUTED]->(cq)
CREATE (ti)-[:WITH_PARAMETERS]->(ps)
```

#### D. Results Metadata
```cypher
CREATE (rs:ResultSummary {
  recordCount: 15,
  executionTimeMs: 245,
  success: true,
  riskScoresFound: [8.5, 7.2, 9.1],
  maxRiskScore: 9.1,
  fraudRingsDetected: 3
})

CREATE (ti)-[:RETURNED]->(rs)
```

#### E. Output Format Tracking
```cypher
MATCH (ti:ToolInvocation)
MERGE (of:OutputFormat {format: "bloom_saved_search"})
CREATE (ti)-[:OUTPUT_FORMAT]->(of)
```

### 5. Analytics Queries (Examples)

#### Most Common Questions
```cypher
MATCH (q:Question)
WITH q.text AS question, count(*) AS frequency
ORDER BY frequency DESC
LIMIT 20
RETURN question, frequency
```

#### Most Used Tools
```cypher
MATCH (ti:ToolInvocation)-[:USED_TOOL]->(t:Tool)
WITH t.name AS tool, count(*) AS invocations
ORDER BY invocations DESC
RETURN tool, invocations
```

#### Tool Usage Workflows (Sequential Patterns)
```cypher
MATCH (s:Session)-[:ASKED]->(q1:Question)-[:MAPPED_TO]->(ti1:ToolInvocation)-[:USED_TOOL]->(t1:Tool)
MATCH (s)-[:ASKED]->(q2:Question)-[:MAPPED_TO]->(ti2:ToolInvocation)-[:USED_TOOL]->(t2:Tool)
WHERE ti1.selectedAt < ti2.selectedAt
WITH t1.name AS firstTool, t2.name AS secondTool, count(*) AS frequency
ORDER BY frequency DESC
RETURN firstTool, secondTool, frequency
```

#### Output Format Preferences
```cypher
MATCH (ti:ToolInvocation)-[:OUTPUT_FORMAT]->(of:OutputFormat)
WITH of.format AS format, count(*) AS usage
RETURN format, usage
ORDER BY usage DESC
```

#### Failed Queries (for debugging)
```cypher
MATCH (ti:ToolInvocation)-[:RETURNED]->(rs:ResultSummary)
WHERE rs.success = false
MATCH (ti)-[:USED_TOOL]->(t:Tool)
MATCH (ti)<-[:MAPPED_TO]-(q:Question)
RETURN q.text, t.name, rs.errorMessage, ti.selectedAt
ORDER BY ti.selectedAt DESC
LIMIT 50
```

#### Query Performance Analysis
```cypher
MATCH (ti:ToolInvocation)-[:EXECUTED]->(cq:CypherQuery)
MATCH (ti)-[:RETURNED]->(rs:ResultSummary)
WITH cq.queryHash AS query,
     avg(rs.executionTimeMs) AS avgTime,
     max(rs.executionTimeMs) AS maxTime,
     count(*) AS executions
WHERE avgTime > 1000  // Slow queries
RETURN query, avgTime, maxTime, executions
ORDER BY avgTime DESC
```

## Implementation Approach

### Phase 1: Basic Event Logging
- Log every tool invocation to analytics DB
- Capture: tool name, parameters, timestamp, success/failure
- Simple schema: just ToolInvocation nodes

### Phase 2: Question Harvesting
- Capture original user question text
- Link questions to tool invocations
- Enable "most common questions" queries

### Phase 3: Query Cataloging
- Store executed Cypher queries
- Deduplicate by query hash
- Track which queries are most valuable (by usage frequency)

### Phase 4: Pattern Analysis
- Semantic clustering of similar questions
- Tool workflow patterns (sequential usage)
- Output format preference analysis

### Phase 5: Feedback Loop
- Use harvested data to improve tool descriptions
- Identify gaps (common questions with no good tool)
- Suggest new tools based on usage patterns

## Configuration

### Analytics Service Interface

```go
type AnalyticsService interface {
    // Log a tool invocation event
    LogToolInvocation(ctx context.Context, event ToolInvocationEvent) error

    // Log a user question
    LogUserQuestion(ctx context.Context, question UserQuestionEvent) error

    // Query analytics data
    GetMostCommonQuestions(ctx context.Context, limit int) ([]QuestionFrequency, error)
    GetToolUsageStats(ctx context.Context) ([]ToolStats, error)
    GetWorkflowPatterns(ctx context.Context) ([]WorkflowPattern, error)
}

type ToolInvocationEvent struct {
    InvocationID    string
    SessionID       string
    UserID          string // Optional, may be anonymous
    QuestionText    string
    ToolName        string
    Parameters      map[string]interface{}
    OutputFormat    string
    QueryText       string
    QueryHash       string
    ExecutionTimeMs int64
    RecordCount     int
    Success         bool
    ErrorMessage    string
    Timestamp       time.Time
}

type UserQuestionEvent struct {
    QuestionID   string
    SessionID    string
    UserID       string
    QuestionText string
    TokenCount   int
    Timestamp    time.Time
}
```

### Environment Configuration

```bash
# Analytics database connection
ANALYTICS_ENABLED=true
ANALYTICS_NEO4J_URI=bolt://localhost:7687
ANALYTICS_NEO4J_USERNAME=neo4j
ANALYTICS_NEO4J_PASSWORD=password
ANALYTICS_NEO4J_DATABASE=fraud_analytics

# Analytics options
ANALYTICS_INCLUDE_QUESTIONS=true      # Log user questions
ANALYTICS_INCLUDE_QUERIES=true        # Log Cypher queries
ANALYTICS_INCLUDE_PARAMETERS=true     # Log query parameters
ANALYTICS_INCLUDE_RESULTS=false       # Don't log actual result data (privacy)
```

## Privacy & Security Considerations

1. **PII Handling** - User questions may contain sensitive info (account IDs, names)
2. **Result Data** - Don't log actual fraud investigation results, only metadata
3. **Access Control** - Analytics DB may be accessible to data scientists, not just fraud analysts
4. **Retention** - Define retention policy (e.g., 90 days for raw events, forever for aggregates)
5. **Anonymization** - Option to hash/anonymize user IDs and account IDs in analytics

## Benefits

### Immediate
- **Usage visibility** - Which tools are valuable vs. unused
- **Performance monitoring** - Identify slow queries
- **Error tracking** - Debug tool failures

### Medium-term
- **Tool improvement** - Refine tool descriptions based on actual usage
- **Gap identification** - Find questions that no tool handles well
- **Workflow optimization** - Understand investigation patterns

### Long-term
- **Query catalog** - Library of battle-tested fraud queries
- **Tool orchestration** - Data to train agentic workflows
- **Product roadmap** - Data-driven decisions on new capabilities
- **Industry benchmarking** - Understand common fraud patterns across users

## Future: Semantic Search & Recommendation

With harvested data, enable:

1. **"Other analysts asked..."** - Suggest similar questions
2. **Query recommendation** - "Users who ran this query also ran..."
3. **Tool suggestion** - Proactively suggest tools based on session context
4. **Saved query library** - Searchable catalog of proven queries
5. **Investigation templates** - Common workflows captured as playbooks

## Next Steps

1. Design detailed analytics schema (separate design doc)
2. Implement `AnalyticsService` interface
3. Add analytics hooks to fraud tool handlers
4. Create analytics dashboard queries
5. Set up analytics DB constraints/indexes
6. Define retention and privacy policies
7. Build query catalog UI (separate web application)
