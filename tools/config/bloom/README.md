# Neo4j Bloom Guidance Tools

This directory contains guidance tools for creating Neo4j Bloom Scene Actions and Search Phrases.

## Tools Overview

### 1. `generate-scene-action`
**Purpose**: Guides users through creating Scene Actions for Neo4j Bloom

**What are Scene Actions?**
- Context-sensitive Cypher queries that use currently selected graph elements as input
- Execute against selected nodes or relationships in Bloom visualization
- Appear in right-click context menu when elements are selected
- Support both READ and WRITE operations

**Use this tool when users want to:**
- Create queries that expand or analyze currently selected nodes/relationships
- Build context-sensitive actions in Bloom
- Perform operations on selected graph elements

**Example Use Cases:**
- "Show all customers who transacted with selected accounts"
- "Find other customers sharing identity attributes with selected customer"
- "Calculate transaction volume for selected accounts"

---

### 2. `generate-search-phrase`
**Purpose**: Guides users through creating Search Phrases for Neo4j Bloom

**What are Search Phrases?**
- Saved, pre-defined queries invoked by typing in the Bloom search bar
- Can be static (fixed query) or dynamic (with user-provided parameters)
- Support parameter suggestions from database or custom queries
- Match case-insensitively with autocomplete

**Use this tool when users want to:**
- Create queries that start fresh investigations
- Build parameterized searches with user input
- Enable natural language search in Bloom

**Example Use Cases:**
- "Customers from $country with transactions over $amount"
- "Transactions between $startDate and $endDate"
- "Connection between $customer1 and $customer2"

---

## Tool Design

Both tools are **documentation-only** guidance tools:
- ✅ No query execution
- ✅ No input parameters required
- ✅ Return comprehensive guidance in description field
- ✅ Guide users through testing and validation workflow

## Workflow Pattern

Both tools follow this conversational workflow:

1. **Understand Requirements**: User describes what they want to find/do
2. **Get Schema**: Tool instructs LLM to call `get-schema` first
3. **Generate Test Query**: Tool guides LLM to create Cypher query
4. **Execute & Validate**: User tests query with real data
5. **Format for Bloom**: Tool provides Bloom-specific configuration
6. **Import Instructions**: Tool explains how to add to Bloom

## Key Differences: Scene Actions vs Search Phrases

| Feature | Scene Actions | Search Phrases |
|---------|---------------|----------------|
| **Invocation** | Right-click context menu | Type in search bar |
| **Input** | Selected graph elements | User-typed parameters |
| **Use case** | Expand/analyze selection | Start new investigation |
| **Parameters** | `$nodes` or `$relationships` | `$userParameter` |
| **When to use** | Exploring from known entities | Finding initial entities |

## Example Conversations

### Scene Action Example

**User**: "I want to create a scene action that shows all high-value transactions for a selected account."

**Assistant** (using `generate-scene-action` tool):
1. Calls `get-schema` to understand database structure
2. Generates test query:
   ```cypher
   MATCH (a:Account)
   WHERE elementId(a) IN $nodes
   MATCH (a)-[t:TRANSACTION]->(dest:Account)
   WHERE t.amount > 10000
   RETURN a, t, dest
   ORDER BY t.amount DESC
   LIMIT 50
   ```
3. User tests query
4. Provides Scene Action configuration with import instructions

---

### Search Phrase Example

**User**: "I want to search for customers from a specific country who have made transactions over a certain amount."

**Assistant** (using `generate-search-phrase` tool):
1. Calls `get-schema` to understand database structure
2. Proposes phrase: "Customers from $country with transactions over $amount"
3. Generates test query with parameters
4. User validates results
5. Provides Search Phrase configuration with parameter suggestions
6. Provides import instructions for Bloom UI

---

## Design Philosophy

These tools prioritize:
- **Testing-first approach**: Always test queries before Bloom formatting
- **Schema awareness**: Always call `get-schema` first
- **Clear workflow**: Step-by-step guidance through the process
- **Practical examples**: Multiple real-world use case patterns
- **Import instructions**: Clear guidance on adding to Bloom

## Resources

- [Neo4j Bloom Scene Actions Docs](https://neo4j.com/docs/bloom-user-guide/current/bloom-tutorial/scene-actions/)
- [Neo4j Bloom Search Phrases Docs](https://neo4j.com/docs/bloom-user-guide/current/bloom-tutorial/search-phrases-advanced/)
- [Cypher Query Language Reference](https://neo4j.com/docs/cypher-manual/current/)
