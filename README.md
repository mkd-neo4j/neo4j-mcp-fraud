# Neo4j Fraud MCP

Model Context Protocol (MCP) server for Neo4j, specialized for fraud detection and financial crime investigation.

## Why Neo4j Fraud MCP?

The Neo4j Fraud MCP server is purpose-built for financial crime analysts, AML investigators, and compliance officers. It combines Neo4j's powerful graph database capabilities with specialized fraud detection tools to:

- **Accelerate investigations** through natural language queries to complex graph patterns
- **Lower the barrier to entry** for analysts who may not be Cypher experts
- **Leverage Neo4j's strengths** in pattern matching, deep traversals, and relationship analysis

### Target Users

- **Fraud Analysts**: Investigating suspicious transactions and customer behavior
- **AML Investigators**: Conducting anti-money laundering investigations
- **KYC/CDD Officers**: Performing know-your-customer and customer due diligence checks
- **Compliance Teams**: Monitoring for regulatory compliance

### Key Use Cases

1. **Transaction Monitoring**: Detecting unusual transaction patterns, circular flows, velocity anomalies
2. **KYC/CDD Compliance**: Validating documentation completeness, identifying missing/expired documents
3. **Network Analysis**: Finding connections to known bad actors, PEPs (Politically Exposed Persons)
4. **Device & Identity Fraud**: Detecting shared devices, credentials, and addresses
5. **Risk Assessment**: Evaluating exposure to high-risk jurisdictions and entities
6. **Synthetic Identity Detection**: Identifying fraudulent identity patterns and suspicious account behavior

## Quick Start

📦 **Installation & Setup**: See [SETUP.md](SETUP.md) for complete installation and configuration instructions

## Architecture: YAML-Based Dynamic Tools

The Neo4j Fraud MCP server uses a **config-based architecture** where most tools are defined as YAML files rather than hardcoded Go implementations. This design provides:

- ✅ **Easy extensibility**: Add new tools by creating YAML files—no Go code changes required
- ✅ **LLM-optimized**: Tool descriptions are embedded in config, providing detailed guidance for AI agents
- ✅ **Clear categorization**: Tools are organized by folder structure (fraud/, bloom/, sar/, graph-data/)
- ✅ **Zero-code customization**: Modify tool behavior, thresholds, and guidance without rebuilding

### Tool Discovery

At startup, the server automatically:
1. Scans the `tools/config/` directory (embedded in binary)
2. Loads all `*.yaml` tool definitions
3. Registers them as MCP tools with their specifications

**Example YAML tool structure:**
```yaml
name: detect-synthetic-identity
title: Detect Synthetic Identity Fraud
description: |
  Detailed guidance for LLMs on how to detect fraud patterns...

input_schema:
  type: object
  properties:
    query:
      type: string
      description: Your Cypher query to detect synthetic identity fraud

execution:
  mode: read  # or 'write'
  timeout: 30000

metadata:
  readonly: true
  category: fraud  # derived from folder: tools/config/fraud/
```

📂 **See [tools/config/README.md](tools/config/README.md)** for complete YAML format documentation and guidelines on creating custom tools.

## Tools & Usage

The server provides two types of tools:

### Infrastructure Tools (Hardcoded in Go)

Core system operations that handle database interaction and metadata:

| Tool                  | ReadOnly | Purpose                                              | Notes                                                                                                                          |
| --------------------- | -------- | ---------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------ |
| `get-schema`          | `true`   | Introspect labels, relationship types, property keys | Provide valuable context to the client LLMs.                                                                                   |
| `read-cypher`         | `true`   | Execute arbitrary Cypher (read mode)                 | Rejects writes, schema/admin operations, and PROFILE queries. Use `write-cypher` instead.                                      |
| `write-cypher`        | `false`  | Execute arbitrary Cypher (write mode)                | **Caution:** LLM-generated queries could cause harm. Use only in development environments. Disabled if `NEO4J_READ_ONLY=true`. |
| `list-gds-procedures` | `true`   | List GDS procedures available in the Neo4j instance  | Help the client LLM to have a better visibility on the GDS procedures available                                                |

### Config-Based Tools (Defined in YAML)

Specialized investigation and guidance tools loaded from `tools/config/`:

| Tool                           | Category    | Purpose                                                 | Config File                                     |
| ------------------------------ | ----------- | ------------------------------------------------------- | ----------------------------------------------- |
| `detect-synthetic-identity`    | fraud       | Detect synthetic identity fraud patterns                | `fraud/detect-synthetic-identity.yaml`          |
| `get-customer-profile`         | graph-data  | Retrieve comprehensive customer profiles               | `graph-data/get-customer-profile.yaml`          |
| `get-transaction-history`      | graph-data  | Retrieve transaction history with filters               | `graph-data/get-transaction-history.yaml`       |
| `get-sar-report-guidance`      | sar         | Suspicious Activity Report filing guidance              | `sar/get-sar-guidance.yaml`                     |
| `generate-scene-action`        | bloom       | Guide for creating Neo4j Bloom Scene Actions            | `bloom/generate-scene-action.yaml`              |
| `generate-search-phrase`       | bloom       | Guide for creating Neo4j Bloom Search Phrases           | `bloom/generate-search-phrase.yaml`             |

**Adding Custom Tools**: Create a new YAML file in the appropriate category folder under `tools/config/`. The server will automatically discover and register it on next startup. See [tools/config/README.md](tools/config/README.md) for the YAML format specification.

### Readonly mode flag

Enable readonly mode by setting the `NEO4J_READ_ONLY` environment variable to `true` (for example, `"NEO4J_READ_ONLY": "true"`). Accepted values are `true` or `false` (default: `false`).

You can also override this setting using the `--neo4j-read-only` CLI flag:

```bash
neo4j-fraud-mcp --neo4j-uri "bolt://localhost:7687" --neo4j-username "neo4j" --neo4j-password "password" --neo4j-read-only true
```

When enabled, write tools (for example, `write-cypher`) are not exposed to clients.

### Query Classification

The `read-cypher` tool performs an extra round-trip to the Neo4j database to guarantee read-only operations.

Important notes:

- **Write operations**: `CREATE`, `MERGE`, `DELETE`, `SET`, etc., are treated as non-read queries.
- **Admin queries**: Commands like `SHOW USERS`, `SHOW DATABASES`, etc., are treated as non-read queries and must use `write-cypher` instead.
- **Profile queries**: `EXPLAIN PROFILE` queries are treated as non-read queries, even if the underlying statement is read-only.
- **Schema operations**: `CREATE INDEX`, `DROP CONSTRAINT`, etc., are treated as non-read queries.

## Example Natural Language Prompts

Below are some example prompts you can try in Copilot or any other MCP client:

### General Database Exploration
- "What does my Neo4j instance contain? List all node labels, relationship types, and property keys."
- "Find all Person nodes and their relationships in my Neo4j instance."

### Fraud Detection Examples
- "Detect synthetic identity fraud patterns for customer ID 12345"
- "Find all accounts that share the same device or IP address with account ABC123"
- "Show me circular transaction flows involving account XYZ789"
- "Identify accounts connected to known fraudsters within 2 hops"
- "Find customers with missing or expired KYC documents"

## Security Best Practices

- Use a restricted Neo4j user for exploration.
- Review generated Cypher before executing in production databases.
- Enable read-only mode (`NEO4J_READ_ONLY=true`) for production environments to prevent accidental data modification.

## Documentation

📦 **[Setup Guide](SETUP.md)** – Installation, configuration, transport modes, TLS, logging, and telemetry
📘 **[Client Setup Guide](docs/CLIENT_SETUP.md)** – Configure VSCode, Claude Desktop, and other MCP clients
📂 **[Tool Configuration](tools/config/README.md)** – Create custom YAML-based tools
📚 **[Contributing Guide](CONTRIBUTING.md)** – Development workflow, testing, and contributions

Issues / feedback: open a GitHub issue with reproduction details (omit sensitive data).
