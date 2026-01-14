package synthetic_identity

import "github.com/mark3labs/mcp-go/mcp"

type PIIRelationship struct {
	RelationshipType     string `json:"relationshipType" jsonschema:"description=The relationship type connecting Customer to PII (e.g. HAS_EMAIL)"`
	TargetLabel          string `json:"targetLabel" jsonschema:"description=The node label of the PII entity (e.g. Email)"`
	IdentifierProperty   string `json:"identifierProperty" jsonschema:"description=The property containing the identifier value (e.g. address for Email)"`
}

type DetectSyntheticIdentityInput struct {
	CustomerId          string            `json:"customerId,omitempty" jsonschema:"description=Optional: Customer ID to investigate. If provided, finds customers sharing PII with this specific customer. If omitted, discovers all clusters of customers sharing PII."`
	PIIRelationships    []PIIRelationship `json:"piiRelationships" jsonschema:"description=Array of PII relationship configurations discovered from the schema. Use get-schema to discover these first."`
	MinSharedAttributes int               `json:"minSharedAttributes,omitempty" jsonschema:"default=2,description=Minimum number of shared identity attributes to flag as suspicious"`
	Limit               int               `json:"limit,omitempty" jsonschema:"default=20,description=Maximum number of results to return (discovery mode) or customers to find (investigation mode)"`
}

// Spec returns the MCP tool specification for synthetic identity fraud detection
func Spec() mcp.Tool {
	return mcp.NewTool("detect-synthetic-identity",
		mcp.WithDescription(`Detects potential synthetic identity fraud by identifying customers who share multiple identity attributes (PII). Operates in two modes:

**MODE 1 - Discovery Mode (customerId omitted):**
Discovers all clusters of customers sharing PII across the database. Use this to find fraud patterns proactively.
Example: "Show me 20 people with shared PII information"

**MODE 2 - Investigation Mode (customerId provided):**
Finds customers sharing PII with a specific target customer. Use this for targeted fraud investigation.
Example: "For customer CUS123, find any other customers related via shared PII"

**REQUIRED WORKFLOW - Schema Discovery:**
This tool is schema-aware and requires you to discover the database structure first:

1. **Call get-schema tool** to retrieve the database schema
2. **Analyze the Customer node** to find its PII relationships (e.g., HAS_EMAIL, HAS_PHONE, HAS_SSN, HAS_PASSPORT, HAS_DRIVER_LICENSE)
3. **For each PII relationship**, identify:
   - relationshipType: The relationship name (e.g., "HAS_EMAIL")
   - targetLabel: The connected node label (e.g., "Email")
   - identifierProperty: The property containing the identifier (e.g., "address" for Email, "number" for Phone/SSN)
4. **Pass discovered relationships** to this tool's piiRelationships parameter

**Example piiRelationships structure:**
[
  {
    "relationshipType": "HAS_EMAIL",
    "targetLabel": "Email",
    "identifierProperty": "address"
  },
  {
    "relationshipType": "HAS_PHONE",
    "targetLabel": "Phone",
    "identifierProperty": "number"
  },
  {
    "relationshipType": "HAS_SSN",
    "targetLabel": "SSN",
    "identifierProperty": "number"
  }
]

**When to use this tool:**
- Discovering fraud patterns proactively (discovery mode)
- Investigating suspected synthetic identity fraud (investigation mode)
- Validating customer identity during onboarding
- Finding fraud rings using fabricated or stolen identity information

**What it detects:**
- Customers sharing any PII attributes
- Clusters of accounts with overlapping identity attributes
- Patterns indicating synthetic or stolen identity use

**Fraud indicators this reveals:**
- CRITICAL: Multiple customers sharing 3+ identity attributes (likely organized fraud ring)
- HIGH RISK: Multiple customers sharing 2+ identity attributes (synthetic identity pattern)
- MEDIUM RISK: Shared single identity attribute (may be legitimate family/business)

**Investigation workflow:**
1. Call get-schema to discover available PII relationships
2. For discovery: Run without customerId to find all PII sharing clusters
3. For investigation: Run with customerId to investigate specific customer
4. Examine the returned clusters of customers with shared attributes
5. Investigate transaction patterns of linked customers
6. Follow up with additional fraud detection tools on connected customers

**Returns:**
- List of customers sharing identity attributes
- Details of which specific attributes are shared (with type and value)
- Count of shared attributes per customer connection`),
		mcp.WithInputSchema[DetectSyntheticIdentityInput](),
		mcp.WithTitleAnnotation("Detect Synthetic Identity Fraud"),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(true),
	)
}
