package customer_profile

import (
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mkd-neo4j/neo4j-mcp-fraud/internal/tools/cypher/query_builder"
)

// EntityConfig defines the configuration for the entity node to retrieve
type EntityConfig struct {
	// NodeLabel is the label of the entity node (e.g., "Customer", "Person", "Account")
	NodeLabel string `json:"nodeLabel" jsonschema:"description=Node label of the entity (e.g. Customer, Person, Account)"`

	// IdProperty is the property name containing the unique identifier (e.g., "customerId", "personId")
	IdProperty string `json:"idProperty" jsonschema:"description=Property name for unique identifier (e.g. customerId, personId)"`

	// BaseProperties are the properties from the entity node to include in base details.
	// If empty, all properties will be returned using properties() function.
	BaseProperties []string `json:"baseProperties,omitempty" jsonschema:"description=List of base properties to include (e.g. [firstName, lastName, dateOfBirth]). If empty, returns all properties."`
}

// GetCustomerProfileInput defines the input parameters for the get-customer-profile tool
type GetCustomerProfileInput struct {
	// EntityId is the unique identifier for the entity (required)
	EntityId string `json:"entityId" jsonschema:"description=Entity ID to retrieve profile for (required)"`

	// EntityConfig defines the entity node configuration
	EntityConfig EntityConfig `json:"entityConfig" jsonschema:"description=Configuration for the entity node (node label, ID property, base properties)"`

	// AttributeMappings defines which attributes to retrieve based on the actual schema.
	// Discovered via get-schema tool.
	AttributeMappings []query_builder.AttributeMapping `json:"attributeMappings" jsonschema:"description=Array of attribute mappings discovered from the schema. Use get-schema to discover these first."`
}

// Spec returns the MCP tool specification for get-customer-profile
func Spec() mcp.Tool {
	return mcp.NewTool("get-customer-profile",
		mcp.WithDescription(`Retrieves comprehensive customer profile information from the graph database.

**SCHEMA-AWARE DESIGN:**
This tool dynamically adapts to your database schema. It does NOT make assumptions about relationship names, node labels, or property names.

**REQUIRED WORKFLOW:**
1. **Call get-schema** to discover your database structure
2. **Analyze the Customer node** to identify attribute relationships (e.g., HAS_EMAIL, HAS_PHONE, HAS_SSN, HAS_ADDRESS, HAS_DRIVER_LICENSE)
3. **For each attribute**, construct an AttributeMapping with:
   - relationshipType: The relationship name from your schema
   - targetLabel: The connected node label from your schema
   - identifierProperty: The property containing the key identifier (e.g., "address" for Email, "number" for Phone/SSN)
   - attributeCategory: Logical grouping ("contact_information", "identity_documents", "employment_details", "account_information")
   - includeProperties: Optional list of specific properties to retrieve
4. **Pass discovered mappings** to this tool's attributeMappings parameter

**EXAMPLE ATTRIBUTE MAPPINGS:**

For identity attributes, accounts, and relationships:
[
  {
    "relationshipType": "HAS_EMAIL",
    "targetLabel": "Email",
    "identifierProperty": "address",
    "attributeCategory": "contact_information",
    "includeProperties": ["verified", "createdAt"]
  },
  {
    "relationshipType": "HAS_PHONE",
    "targetLabel": "Phone",
    "identifierProperty": "number",
    "attributeCategory": "contact_information",
    "includeProperties": ["type", "primary"]
  },
  {
    "relationshipType": "HAS_SSN",
    "targetLabel": "SSN",
    "identifierProperty": "number",
    "attributeCategory": "identity_documents",
    "includeProperties": ["issuedDate"]
  },
  {
    "relationshipType": "HAS_ADDRESS",
    "targetLabel": "Address",
    "identifierProperty": null,
    "attributeCategory": "contact_information",
    "includeProperties": ["street", "city", "state", "zip", "country", "type", "validFrom", "validTo"]
  },
  {
    "relationshipType": "HAS_DRIVER_LICENSE",
    "targetLabel": "DriverLicense",
    "identifierProperty": "number",
    "attributeCategory": "identity_documents",
    "includeProperties": ["state", "expiryDate"]
  },
  {
    "relationshipType": "OWNS",
    "targetLabel": "Account",
    "identifierProperty": "accountNumber",
    "attributeCategory": "account_information",
    "includeProperties": ["accountType", "openedDate", "status", "balance"]
  },
  {
    "relationshipType": "BENEFICIAL_OWNER_OF",
    "targetLabel": "Entity",
    "identifierProperty": "entityId",
    "attributeCategory": "relationships",
    "includeProperties": ["name", "type"]
  }
]

**WHEN TO USE THIS TOOL:**
- Gathering customer identity details for SAR Section 1.3 (Subject Identity Details)
- Collecting customer information for KYC/CDD compliance
- Creating comprehensive customer profiles for investigations
- Verifying identity document completeness
- General customer data retrieval for any compliance or analysis purpose

**USE CASES:**
- **Fraud Investigation:** Gather complete customer profile for SAR filing
- **KYC/CDD:** Verify customer identity and documentation
- **Compliance:** Audit customer information completeness
- **Customer Service:** Retrieve full customer details for support
- **Data Analytics:** Extract customer demographics for analysis

**OUTPUT STRUCTURE:**
Returns a structured customer profile organized by attribute categories:
- base_details: firstName, lastName, dateOfBirth, nationality, etc. (from entity node base properties)
- contact_information: emails, phones, addresses (if mapped via AttributeMappings)
- identity_documents: SSN, driver license, passport, etc. (if mapped via AttributeMappings)
- employment_details: occupation, employer, business type (if mapped via AttributeMappings)
- account_information: accounts owned by the entity (if mapped via AttributeMappings with category "account_information")
- relationships: beneficial owners, authorized users, etc. (if mapped via AttributeMappings with category "relationships")

All categories are determined by the attributeCategory field in your AttributeMappings.

**EXAMPLE USAGE:**
User: "Get the complete profile for customer CUS123 including all identity documents"
LLM: [calls get-schema, analyzes Customer relationships, constructs AttributeMappings, calls get-customer-profile]

**IMPORTANT NOTES:**
- This tool uses OPTIONAL MATCH, so missing relationships will return empty arrays (not errors)
- All attribute categories are optional - only include what exists in your schema
- The tool is generic and works for ANY Neo4j graph schema with Customer nodes
- Not fraud-specific - suitable for KYC, compliance, analytics, and general data retrieval`),
		mcp.WithInputSchema[GetCustomerProfileInput](),
		mcp.WithTitleAnnotation("Get Customer Profile"),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(true),
	)
}
