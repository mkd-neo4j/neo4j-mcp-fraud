# Dynamic Config-Based Tools

This directory contains YAML configurations for dynamically loaded MCP tools.

## Directory Structure

```
tools/config/
├── bloom/                  # Neo4j Bloom guidance tools
│   ├── generate-scene-action.yaml
│   └── generate-search-phrase.yaml
├── fraud/                  # Fraud detection tools
│   └── detect-synthetic-identity.yaml
├── graph-data/             # Data retrieval tools
│   ├── get-customer-profile.yaml
│   └── get-transaction-history.yaml
├── liquidity/              # Liquidity analysis tools
│   ├── analyze-account-fund-flow.yaml
│   ├── analyze-merchant-revenue-flows.yaml
│   └── identify-liquidity-concentration.yaml
└── sar/                    # SAR-related tools
    └── get-sar-guidance.yaml
```

## When to Use Config-Based Tools

Config-based tools (YAML) are ideal for:
- ✅ Tools that execute **Cypher queries** against Neo4j
- ✅ Tools where **LLMs can write better queries** than hardcoded logic
- ✅ **Specialized query patterns** with detailed guidance
- ✅ Tools that benefit from **easy customization** without code changes
- ✅ **Documentation-only guidance tools** that provide instructions without executing queries

## When to Keep Hardcoded Tools

Hardcoded tools (Go code) are better for:
- ❌ **Infrastructure tools** (read-cypher, write-cypher, get-schema, list-gds-procedures)
  - Core system operations that don't fit the config-based pattern
- ❌ **Static reference documentation** tools that return embedded markdown/docs
  - get-neo4j-reference-data-models (loads Neo4j data model docs)
- ❌ **Complex orchestration** logic that goes beyond query execution
- ❌ Tools with **complex parameter marshalling** or validation

**Updated Rule**: Documentation-only guidance tools (SAR guidance, Bloom guidance) work well as config-based tools since they only need a description field.

## YAML Tool Format

Each config-based tool requires:

```yaml
name: tool-name
title: Human Readable Title
description: |
  Detailed guidance for LLMs on how to use this tool.
  Include example Cypher queries and patterns.

input_schema:
  type: object
  required: [query]
  properties:
    query:
      type: string
      description: Your Cypher query to execute
    params:
      type: object
      description: Optional query parameters

execution:
  mode: read  # or 'write'
  timeout: 30000  # milliseconds

metadata:
  readonly: true
  idempotent: true
  destructive: false
  category: fraud  # derived from folder name
```

## Current Status

### Config-Based Tools (Dynamic)

#### Bloom Guidance Tools
- ✅ `generate-scene-action` - Guide for creating Neo4j Bloom Scene Actions (documentation-only)
- ✅ `generate-search-phrase` - Guide for creating Neo4j Bloom Search Phrases (documentation-only)

#### Fraud Detection Tools
- ✅ `detect-synthetic-identity` - Finds entities sharing PII attributes

#### Graph Data Retrieval Tools
- ✅ `get-customer-profile` - Retrieves comprehensive customer profiles
- ✅ `get-transaction-history` - Retrieves transaction history with filters

#### SAR & Compliance Tools
- ✅ `get-sar-report-guidance` - Suspicious Activity Report filing guidance (documentation-only)

#### Liquidity Analysis Tools
- ✅ `analyze-account-fund-flow` - Analyzes inflows/outflows for an account
- ✅ `identify-liquidity-concentration` - Finds high-volume accounts (liquidity hubs)
- ✅ `analyze-merchant-revenue-flows` - Analyzes payment flows to merchants/banks

### Hardcoded Tools (Go)
- `get-schema` - Schema introspection
- `read-cypher` - Execute read-only Cypher
- `write-cypher` - Execute write Cypher
- `list-gds-procedures` - GDS discovery
- `get-neo4j-reference-data-models` - Data model reference (static content)

## Adding New Config-Based Tools

1. Create a new YAML file in the appropriate category folder
2. Follow the YAML format above
3. Provide detailed LLM guidance with example queries
4. Rebuild the server - tools are auto-discovered on startup
5. No Go code changes required!

## Migration Strategy

When migrating existing tools to config-based:
1. Create YAML config with equivalent functionality
2. Test with LLM-generated queries
3. Compare results with original tool
4. If successful, remove old Go implementation
5. Update server instructions and documentation
