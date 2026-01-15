package customer_profile

import (
	"strings"
	"testing"

	"github.com/mkd-neo4j/neo4j-mcp-fraud/internal/tools/cypher/query_builder"
	"github.com/stretchr/testify/assert"
)

var testEntityConfig = EntityConfig{
	NodeLabel:      "Customer",
	IdProperty:     "customerId",
	BaseProperties: []string{"firstName", "lastName", "dateOfBirth"},
}

func TestBuildCustomerProfileQuery_BasicContactInformation(t *testing.T) {
	mappings := []query_builder.AttributeMapping{
		{
			RelationshipType:   "HAS_EMAIL",
			TargetLabel:        "Email",
			IdentifierProperty: "address",
			AttributeCategory:  "contact_information",
			IncludeProperties:  []string{"verified", "createdAt"},
		},
	}

	query := buildCustomerProfileQuery(testEntityConfig, mappings)

	// Verify query structure
	assert.Contains(t, query, "MATCH (e:Customer {customerId: $entityId})")
	assert.Contains(t, query, "OPTIONAL MATCH (e)-[:HAS_EMAIL]->")
	assert.Contains(t, query, ":Email")
	assert.Contains(t, query, "WITH e")
	assert.Contains(t, query, "collect(DISTINCT")
	assert.Contains(t, query, "base_details")
	assert.Contains(t, query, "contact_information")
	assert.Contains(t, query, "emails:")
	// Should use map projection syntax in WITH clause
	assert.Contains(t, query, "attr0{.address, .verified, .createdAt}")
}

func TestBuildCustomerProfileQuery_MultipleIdentityDocuments(t *testing.T) {
	mappings := []query_builder.AttributeMapping{
		{
			RelationshipType:   "HAS_SSN",
			TargetLabel:        "SSN",
			IdentifierProperty: "number",
			AttributeCategory:  "identity_documents",
			IncludeProperties:  []string{"issuedDate"},
		},
		{
			RelationshipType:   "HAS_DRIVER_LICENSE",
			TargetLabel:        "DriverLicense",
			IdentifierProperty: "number",
			AttributeCategory:  "identity_documents",
			IncludeProperties:  []string{"state", "expiryDate"},
		},
	}

	query := buildCustomerProfileQuery(testEntityConfig, mappings)

	// Verify both identity documents are included
	assert.Contains(t, query, "OPTIONAL MATCH (e)-[:HAS_SSN]->")
	assert.Contains(t, query, "OPTIONAL MATCH (e)-[:HAS_DRIVER_LICENSE]->")
	assert.Contains(t, query, "identity_documents")
	assert.Contains(t, query, "ssns:")
	assert.Contains(t, query, "driverlicenses:")
}

func TestBuildCustomerProfileQuery_WithAccounts(t *testing.T) {
	mappings := []query_builder.AttributeMapping{
		{
			RelationshipType:   "HAS_EMAIL",
			TargetLabel:        "Email",
			IdentifierProperty: "address",
			AttributeCategory:  "contact_information",
		},
		{
			RelationshipType:   "OWNS",
			TargetLabel:        "Account",
			IdentifierProperty: "accountNumber",
			AttributeCategory:  "account_information",
			IncludeProperties:  []string{"accountType", "openedDate", "status", "balance"},
		},
	}

	query := buildCustomerProfileQuery(testEntityConfig, mappings)

	// Verify accounts are included via AttributeMappings
	assert.Contains(t, query, "OPTIONAL MATCH (e)-[:OWNS]->")
	assert.Contains(t, query, ":Account")
	assert.Contains(t, query, "account_information")
	assert.Contains(t, query, "accounts:")
	// Should use map projection syntax - variable assignment depends on order in query
	assert.Contains(t, query, "{.accountNumber, .accountType, .openedDate, .status, .balance}")
}

func TestBuildCustomerProfileQuery_WithRelationships(t *testing.T) {
	mappings := []query_builder.AttributeMapping{
		{
			RelationshipType:   "HAS_EMAIL",
			TargetLabel:        "Email",
			IdentifierProperty: "address",
			AttributeCategory:  "contact_information",
		},
		{
			RelationshipType:   "BENEFICIAL_OWNER_OF",
			TargetLabel:        "Entity",
			IdentifierProperty: "entityId",
			AttributeCategory:  "relationships",
			IncludeProperties:  []string{"name", "type"},
		},
	}

	query := buildCustomerProfileQuery(testEntityConfig, mappings)

	// Verify relationships are included via AttributeMappings
	assert.Contains(t, query, "OPTIONAL MATCH (e)-[:BENEFICIAL_OWNER_OF]->")
	assert.Contains(t, query, ":Entity")
	assert.Contains(t, query, "relationships")
	assert.Contains(t, query, "entitys:")  // Note: simple pluralization adds 's'
	// Should use map projection syntax - variable depends on order
	assert.Contains(t, query, "{.entityId, .name, .type}")
}

func TestBuildCustomerProfileQuery_CompleteProfile(t *testing.T) {
	mappings := []query_builder.AttributeMapping{
		{
			RelationshipType:   "HAS_EMAIL",
			TargetLabel:        "Email",
			IdentifierProperty: "address",
			AttributeCategory:  "contact_information",
			IncludeProperties:  []string{"verified"},
		},
		{
			RelationshipType:   "HAS_PHONE",
			TargetLabel:        "Phone",
			IdentifierProperty: "number",
			AttributeCategory:  "contact_information",
			IncludeProperties:  []string{"type"},
		},
		{
			RelationshipType:   "HAS_SSN",
			TargetLabel:        "SSN",
			IdentifierProperty: "number",
			AttributeCategory:  "identity_documents",
		},
		{
			RelationshipType:   "OWNS",
			TargetLabel:        "Account",
			IdentifierProperty: "accountNumber",
			AttributeCategory:  "account_information",
			IncludeProperties:  []string{"accountType", "status", "balance"},
		},
		{
			RelationshipType:   "BENEFICIAL_OWNER_OF",
			TargetLabel:        "Entity",
			IdentifierProperty: "entityId",
			AttributeCategory:  "relationships",
			IncludeProperties:  []string{"name", "type"},
		},
	}

	query := buildCustomerProfileQuery(testEntityConfig, mappings)

	// Verify all sections are present
	assert.Contains(t, query, "base_details")
	assert.Contains(t, query, "contact_information")
	assert.Contains(t, query, "identity_documents")
	assert.Contains(t, query, "account_information")
	assert.Contains(t, query, "relationships")

	// Verify structure
	assert.Contains(t, query, "MATCH (e:Customer {customerId: $entityId})")
	assert.Contains(t, query, "RETURN {")
	assert.Contains(t, query, "} as entityProfile")
}

func TestBuildCustomerProfileQuery_MixedCategories(t *testing.T) {
	mappings := []query_builder.AttributeMapping{
		{
			RelationshipType:   "HAS_EMAIL",
			TargetLabel:        "Email",
			IdentifierProperty: "address",
			AttributeCategory:  "contact_information",
		},
		{
			RelationshipType:   "HAS_SSN",
			TargetLabel:        "SSN",
			IdentifierProperty: "number",
			AttributeCategory:  "identity_documents",
		},
		{
			RelationshipType:   "HAS_EMPLOYMENT",
			TargetLabel:        "Employment",
			AttributeCategory:  "employment_details",
			IncludeProperties:  []string{"occupation", "employer"},
		},
	}

	query := buildCustomerProfileQuery(testEntityConfig, mappings)

	// Verify all categories are present
	assert.Contains(t, query, "contact_information")
	assert.Contains(t, query, "identity_documents")
	assert.Contains(t, query, "employment_details")

	// Verify each has the correct relationship
	assert.Contains(t, query, "OPTIONAL MATCH (e)-[:HAS_EMAIL]->")
	assert.Contains(t, query, "OPTIONAL MATCH (e)-[:HAS_SSN]->")
	assert.Contains(t, query, "OPTIONAL MATCH (e)-[:HAS_EMPLOYMENT]->")
}

func TestBuildCategoryReturnClause(t *testing.T) {
	// Test the new approach using pre-collected variables
	collectionAliases := map[string]string{
		"emails": "contact_information_emails",
		"phones": "contact_information_phones",
	}

	clause := buildCategoryReturnClauseFromCollections("contact_information", collectionAliases)

	// Verify structure
	assert.Contains(t, clause, "contact_information: {")
	assert.Contains(t, clause, "emails:")
	assert.Contains(t, clause, "phones:")
	// Should reference pre-collected variables, not use collect()
	assert.Contains(t, clause, "contact_information_emails")
	assert.Contains(t, clause, "contact_information_phones")
	// Should NOT contain collect() since collections are pre-aggregated
	assert.NotContains(t, clause, "collect(")
}

func TestBuildCustomerProfileQuery_NoMappings(t *testing.T) {
	// This should not happen in practice due to validation, but test the builder behavior
	mappings := []query_builder.AttributeMapping{}

	query := buildCustomerProfileQuery(testEntityConfig, mappings)

	// Should still have base query structure
	assert.Contains(t, query, "MATCH (e:Customer {customerId: $entityId})")
	assert.Contains(t, query, "base_details")
	assert.Contains(t, query, "RETURN {")

	// Should not have any OPTIONAL MATCH clauses
	optionalMatchCount := strings.Count(query, "OPTIONAL MATCH")
	assert.Equal(t, 0, optionalMatchCount)
}

func TestBuildCustomerProfileQuery_AllPropertiesMode(t *testing.T) {
	mappings := []query_builder.AttributeMapping{
		{
			RelationshipType:   "HAS_ADDRESS",
			TargetLabel:        "Address",
			AttributeCategory:  "contact_information",
			// No IncludeProperties - should use properties()
		},
	}

	query := buildCustomerProfileQuery(testEntityConfig, mappings)

	// Should use .* map projection for all properties
	assert.Contains(t, query, "attr0{.*}")
	// Note: simple pluralization adds 's' -> "addresss" (Address + s)
	assert.Contains(t, query, "addresss:")
}

func TestBuildCustomerProfileQuery_EnsuresValidCypher(t *testing.T) {
	mappings := []query_builder.AttributeMapping{
		{
			RelationshipType:   "HAS_EMAIL",
			TargetLabel:        "Email",
			IdentifierProperty: "address",
			AttributeCategory:  "contact_information",
		},
		{
			RelationshipType:   "OWNS",
			TargetLabel:        "Account",
			IdentifierProperty: "accountNumber",
			AttributeCategory:  "account_information",
		},
		{
			RelationshipType:   "BENEFICIAL_OWNER_OF",
			TargetLabel:        "Entity",
			IdentifierProperty: "entityId",
			AttributeCategory:  "relationships",
		},
	}

	query := buildCustomerProfileQuery(testEntityConfig, mappings)

	// Verify Cypher syntax essentials
	assert.True(t, strings.HasPrefix(query, "MATCH"))
	assert.Contains(t, query, "RETURN {")
	assert.True(t, strings.HasSuffix(strings.TrimSpace(query), "} as entityProfile"))

	// Verify no syntax errors in structure
	assert.NotContains(t, query, ",,") // No double commas
	assert.NotContains(t, query, "{}") // No empty objects
}

func TestBuildCustomerProfileQuery_BaseDetailsAlwaysFirst(t *testing.T) {
	mappings := []query_builder.AttributeMapping{
		{
			RelationshipType:  "HAS_EMAIL",
			TargetLabel:       "Email",
			AttributeCategory: "contact_information",
		},
	}

	query := buildCustomerProfileQuery(testEntityConfig, mappings)

	// Find RETURN clause
	returnPos := strings.Index(query, "RETURN {")
	assert.True(t, returnPos >= 0, "Should have RETURN clause")

	returnClause := query[returnPos:]

	// Find positions within RETURN clause
	baseDetailsPos := strings.Index(returnClause, "base_details")
	contactInfoPos := strings.Index(returnClause, "contact_information")

	// base_details should come before contact_information in RETURN clause
	assert.True(t, baseDetailsPos < contactInfoPos,
		"base_details should appear before other categories in RETURN clause")
}
