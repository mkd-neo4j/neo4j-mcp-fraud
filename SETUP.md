# Neo4j Fraud MCP - Installation & Setup

Complete guide for installing, configuring, and deploying the Neo4j Fraud MCP server.

## Prerequisites

- A running Neo4j database instance; options include [Aura](https://neo4j.com/product/auradb/), [neo4j–desktop](https://neo4j.com/download/) or [self-managed](https://neo4j.com/deployment-center/#gdb-tab).
- APOC plugin installed in the Neo4j instance.
- Any MCP-compatible client (e.g. [VSCode](https://code.visualstudio.com/) with [MCP support](https://code.visualstudio.com/docs/copilot/customization/mcp-servers))

## Startup Checks & Adaptive Operation

The server performs several pre-flight checks at startup to ensure your environment is correctly configured.

**STDIO Mode - Mandatory Requirements**
In STDIO mode, the server verifies the following core requirements. If any of these checks fail (e.g., due to an invalid configuration, incorrect credentials, or a missing APOC installation), the server will not start:

- A valid connection to your Neo4j instance.
- The ability to execute queries.
- The presence of the APOC plugin.

**HTTP Mode - Verification Skipped**
In HTTP mode, startup verification checks are skipped because credentials come from per-request Basic Auth headers. The server starts immediately without connecting to Neo4j at startup.

**Optional Requirements**
If an optional dependency is missing, the server will start in an adaptive mode. For instance, if the Graph Data Science (GDS) library is not detected in your Neo4j installation, the server will still launch but will automatically disable all GDS-related tools, such as `list-gds-procedures`. All other tools will remain available.

## Installation (Binary)

Releases: https://github.com/mkd-neo4j/neo4j-mcp-fraud/releases

1. Download the archive for your OS/arch.
2. Extract and place `neo4j-fraud-mcp` in a directory present in your PATH variables (see examples below).

Mac / Linux:

```bash
chmod +x neo4j-fraud-mcp
sudo mv neo4j-fraud-mcp /usr/local/bin/
```

Windows (PowerShell / cmd):

```powershell
move neo4j-fraud-mcp.exe C:\Windows\System32
```

Verify the neo4j-fraud-mcp installation:

```bash
neo4j-fraud-mcp -v
```

Should print the installed version.

## Transport Modes

The Neo4j Fraud MCP server supports two transport modes:

- **STDIO** (default): Standard MCP communication via stdin/stdout for desktop clients (Claude Desktop, VSCode)
- **HTTP**: RESTful HTTP server with per-request Basic Authentication for web-based clients and multi-tenant scenarios

### Key Differences

| Aspect               | STDIO                                                      | HTTP                                                                       |
| -------------------- | ---------------------------------------------------------- | -------------------------------------------------------------------------- |
| Startup Verification | Required - server verifies APOC, connectivity, queries     | Skipped - server starts immediately                                        |
| Credentials          | Set via environment variables                              | Per-request via Basic Auth headers                                         |
| Telemetry            | Collects Neo4j version, edition, Cypher version at startup | Reports "unknown-http-mode" - actual version info not available at startup |

See the [Client Setup Guide](docs/CLIENT_SETUP.md) for configuration instructions for both modes.

## TLS/HTTPS Configuration

When using HTTP transport mode, you can enable TLS/HTTPS for secure communication:

### Environment Variables

- `NEO4J_MCP_HTTP_TLS_ENABLED` - Enable TLS/HTTPS: `true` or `false` (default: `false`)
- `NEO4J_MCP_HTTP_TLS_CERT_FILE` - Path to TLS certificate file (required when TLS is enabled)
- `NEO4J_MCP_HTTP_TLS_KEY_FILE` - Path to TLS private key file (required when TLS is enabled)
- `NEO4J_MCP_HTTP_PORT` - HTTP server port (default: `443` when TLS enabled, `80` when TLS disabled)

### Security Configuration

- **Minimum TLS Version**: Hardcoded to TLS 1.2 (allows TLS 1.3 negotiation)
- **Cipher Suites**: Uses Go's secure default cipher suites
- **Default Port**: Automatically uses port 443 when TLS is enabled (standard HTTPS port)

### Example Configuration

```bash
export NEO4J_URI="bolt://localhost:7687"
export NEO4J_MCP_TRANSPORT="http"
export NEO4J_MCP_HTTP_TLS_ENABLED="true"
export NEO4J_MCP_HTTP_TLS_CERT_FILE="/path/to/cert.pem"
export NEO4J_MCP_HTTP_TLS_KEY_FILE="/path/to/key.pem"

neo4j-fraud-mcp
# Server will listen on https://127.0.0.1:443 by default
```

**Production Usage**: Use certificates from a trusted Certificate Authority (e.g., Let's Encrypt, or your organisation) for production deployments.

For detailed instructions on certificate generation, testing TLS, and production deployment, see [CONTRIBUTING.md](CONTRIBUTING.md#tlshttps-configuration).

## Configuration Options

The `neo4j-fraud-mcp` server can be configured using environment variables or CLI flags. CLI flags take precedence over environment variables.

### Environment Variables

See the [Client Setup Guide](docs/CLIENT_SETUP.md) for configuration examples.

### CLI Flags

You can override any environment variable using CLI flags:

```bash
neo4j-fraud-mcp --neo4j-uri "bolt://localhost:7687" \
          --neo4j-username "neo4j" \
          --neo4j-password "password" \
          --neo4j-database "neo4j" \
          --neo4j-read-only false \
          --neo4j-telemetry true
```

Available flags:

- `--neo4j-uri` - Neo4j connection URI (overrides NEO4J_URI)
- `--neo4j-username` - Database username (overrides NEO4J_USERNAME)
- `--neo4j-password` - Database password (overrides NEO4J_PASSWORD)
- `--neo4j-database` - Database name (overrides NEO4J_DATABASE)
- `--neo4j-read-only` - Enable read-only mode: `true` or `false` (overrides NEO4J_READ_ONLY)
- `--neo4j-telemetry` - Enable telemetry: `true` or `false` (overrides NEO4J_TELEMETRY)
- `--neo4j-schema-sample-size` - Modify the sample size used to infer the Neo4j schema
- `--neo4j-transport-mode` - Transport mode: `stdio` or `http` (overrides NEO4J_MCP_TRANSPORT)
- `--neo4j-http-host` - HTTP server host (overrides NEO4J_MCP_HTTP_HOST)
- `--neo4j-http-port` - HTTP server port (overrides NEO4J_MCP_HTTP_PORT)
- `--neo4j-http-tls-enabled` - Enable TLS/HTTPS: `true` or `false` (overrides NEO4J_MCP_HTTP_TLS_ENABLED)
- `--neo4j-http-tls-cert-file` - Path to TLS certificate file (overrides NEO4J_MCP_HTTP_TLS_CERT_FILE)
- `--neo4j-http-tls-key-file` - Path to TLS private key file (overrides NEO4J_MCP_HTTP_TLS_KEY_FILE)

Use `neo4j-fraud-mcp --help` to see all available options.

## Logging

The server uses structured logging with support for multiple log levels and output formats.

### Configuration

**Log Level** (`NEO4J_LOG_LEVEL`, default: `info`)

Controls the verbosity of log output. Supports all [MCP log levels](https://modelcontextprotocol.io/specification/2025-03-26/server/utilities/logging#log-levels): `debug`, `info`, `notice`, `warning`, `error`, `critical`, `alert`, `emergency`.

**Log Format** (`NEO4J_LOG_FORMAT`, default: `text`)

Controls the output format:

- `text` - Human-readable text format (default)
- `json` - Structured JSON format (useful for log aggregation)

## Telemetry

By default, `neo4j-fraud-mcp` collects anonymous usage data to help us improve the product.
This includes information like the tools being used, the operating system, and CPU architecture.
We do not collect any personal or sensitive information.

To disable telemetry, set the `NEO4J_TELEMETRY` environment variable to `"false"`. Accepted values are `true` or `false` (default: `true`).

You can also use the `--neo4j-telemetry` CLI flag to override this setting.

## Client Configuration

To configure MCP clients (VSCode, Claude Desktop, etc.) to use the Neo4j Fraud MCP server:

📘 **[Client Setup Guide](docs/CLIENT_SETUP.md)** – Complete configuration for STDIO and HTTP modes

## Next Steps

Once installed and configured:
- Review the [README.md](README.md) to understand the architecture and available tools
- Explore example prompts in the [README.md](README.md#example-natural-language-prompts)
- For development and contributions, see [CONTRIBUTING.md](CONTRIBUTING.md)
