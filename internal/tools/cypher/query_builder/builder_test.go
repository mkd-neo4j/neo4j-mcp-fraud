package query_builder

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOptionalMatchBuilder_AddAttributeMatch(t *testing.T) {
	builder := NewOptionalMatchBuilder()

	varName := builder.AddAttributeMatch("c", AttributeMapping{
		RelationshipType: "HAS_EMAIL",
		TargetLabel:      "Email",
	})

	assert.Equal(t, "attr0", varName)

	query := builder.Build()
	assert.Contains(t, query, "OPTIONAL MATCH (c)-[:HAS_EMAIL]->(attr0:Email)")
}

func TestOptionalMatchBuilder_AddMultipleMatches(t *testing.T) {
	builder := NewOptionalMatchBuilder()

	var1 := builder.AddAttributeMatch("c", AttributeMapping{
		RelationshipType: "HAS_EMAIL",
		TargetLabel:      "Email",
	})

	var2 := builder.AddAttributeMatch("c", AttributeMapping{
		RelationshipType: "HAS_PHONE",
		TargetLabel:      "Phone",
	})

	assert.Equal(t, "attr0", var1)
	assert.Equal(t, "attr1", var2)

	query := builder.Build()
	assert.Contains(t, query, "OPTIONAL MATCH (c)-[:HAS_EMAIL]->(attr0:Email)")
	assert.Contains(t, query, "OPTIONAL MATCH (c)-[:HAS_PHONE]->(attr1:Phone)")
	assert.Equal(t, 2, builder.GetClauseCount())
}

func TestOptionalMatchBuilder_AddPathMatch_OutDirection(t *testing.T) {
	builder := NewOptionalMatchBuilder()

	varName := builder.AddPathMatch("c", PathSpecification{
		RelationshipType: "KNOWS",
		Direction:        "out",
		TargetLabel:      "Person",
		MinHops:          1,
		MaxHops:          3,
	})

	assert.Equal(t, "path0", varName)

	query := builder.Build()
	assert.Contains(t, query, "OPTIONAL MATCH (c)-[:KNOWS*1..3]->(path0:Person)")
}

func TestOptionalMatchBuilder_AddPathMatch_InDirection(t *testing.T) {
	builder := NewOptionalMatchBuilder()

	varName := builder.AddPathMatch("c", PathSpecification{
		RelationshipType: "FOLLOWS",
		Direction:        "in",
		TargetLabel:      "User",
		MaxHops:          2,
	})

	assert.Equal(t, "path0", varName)

	query := builder.Build()
	assert.Contains(t, query, "OPTIONAL MATCH (c)<-[:FOLLOWS*..2]-(path0:User)")
}

func TestOptionalMatchBuilder_AddPathMatch_BothDirection(t *testing.T) {
	builder := NewOptionalMatchBuilder()

	varName := builder.AddPathMatch("c", PathSpecification{
		RelationshipType: "CONNECTED_TO",
		Direction:        "both",
		TargetLabel:      "Node",
	})

	assert.Equal(t, "path0", varName)

	query := builder.Build()
	assert.Contains(t, query, "OPTIONAL MATCH (c)-[:CONNECTED_TO]-(path0:Node)")
}

func TestOptionalMatchBuilder_AddPathMatch_ExactHops(t *testing.T) {
	builder := NewOptionalMatchBuilder()

	varName := builder.AddPathMatch("c", PathSpecification{
		RelationshipType: "KNOWS",
		Direction:        "out",
		TargetLabel:      "Person",
		MinHops:          2,
		MaxHops:          2,
	})

	assert.Equal(t, "path0", varName)

	query := builder.Build()
	assert.Contains(t, query, "OPTIONAL MATCH (c)-[:KNOWS*2]->(path0:Person)")
}

func TestOptionalMatchBuilder_AddCustomMatch(t *testing.T) {
	builder := NewOptionalMatchBuilder()

	builder.AddCustomMatch("(c)-[:COMPLEX_PATTERN]->(n:Node {status: 'active'})")

	query := builder.Build()
	assert.Contains(t, query, "OPTIONAL MATCH (c)-[:COMPLEX_PATTERN]->(n:Node {status: 'active'})")
}

func TestOptionalMatchBuilder_EmptyBuilder(t *testing.T) {
	builder := NewOptionalMatchBuilder()

	query := builder.Build()
	assert.Equal(t, "", query)
	assert.Equal(t, 0, builder.GetClauseCount())
}

func TestCollectionBuilder_AddProperty(t *testing.T) {
	builder := NewCollectionBuilder()

	builder.AddProperty("email", "e", "address")
	builder.AddProperty("verified", "e", "verified")

	result := builder.Build()
	assert.Equal(t, "{email: e.address, verified: e.verified}", result)
}

func TestCollectionBuilder_AddAllProperties(t *testing.T) {
	builder := NewCollectionBuilder()

	builder.AddAllProperties("props", "n")

	result := builder.Build()
	assert.Equal(t, "{props: properties(n)}", result)
}

func TestCollectionBuilder_AddCustomExpression(t *testing.T) {
	builder := NewCollectionBuilder()

	builder.AddCustomExpression("fullName", "c.firstName + ' ' + c.lastName")
	builder.AddProperty("email", "c", "email")

	result := builder.Build()
	assert.Contains(t, result, "fullName: c.firstName + ' ' + c.lastName")
	assert.Contains(t, result, "email: c.email")
}

func TestCollectionBuilder_BuildDistinctCollection(t *testing.T) {
	builder := NewCollectionBuilder()

	builder.AddProperty("id", "n", "nodeId")

	result := builder.BuildDistinctCollection()
	assert.Equal(t, "collect(DISTINCT {id: n.nodeId})", result)
}

func TestCollectionBuilder_BuildCollection(t *testing.T) {
	builder := NewCollectionBuilder()

	builder.AddProperty("id", "n", "nodeId")

	result := builder.BuildCollection()
	assert.Equal(t, "collect({id: n.nodeId})", result)
}

func TestCollectionBuilder_Empty(t *testing.T) {
	builder := NewCollectionBuilder()

	result := builder.Build()
	assert.Equal(t, "{}", result)
}

func TestGroupMappingsByCategory(t *testing.T) {
	mappings := []AttributeMapping{
		{
			RelationshipType:  "HAS_EMAIL",
			TargetLabel:       "Email",
			AttributeCategory: "contact_information",
		},
		{
			RelationshipType:  "HAS_PHONE",
			TargetLabel:       "Phone",
			AttributeCategory: "contact_information",
		},
		{
			RelationshipType:  "HAS_SSN",
			TargetLabel:       "SSN",
			AttributeCategory: "identity_documents",
		},
		{
			RelationshipType:  "HAS_PASSPORT",
			TargetLabel:       "Passport",
			AttributeCategory: "identity_documents",
		},
		{
			RelationshipType: "HAS_NOTE",
			TargetLabel:      "Note",
			// No category - should go to "other_attributes"
		},
	}

	grouped := GroupMappingsByCategory(mappings)

	assert.Len(t, grouped, 3)
	assert.Len(t, grouped["contact_information"], 2)
	assert.Len(t, grouped["identity_documents"], 2)
	assert.Len(t, grouped["other_attributes"], 1)

	// Check specific mappings
	assert.Equal(t, "Email", grouped["contact_information"][0].TargetLabel)
	assert.Equal(t, "Phone", grouped["contact_information"][1].TargetLabel)
	assert.Equal(t, "SSN", grouped["identity_documents"][0].TargetLabel)
	assert.Equal(t, "Note", grouped["other_attributes"][0].TargetLabel)
}

func TestBuildPropertyMap_WithSpecificProperties(t *testing.T) {
	mapping := AttributeMapping{
		IdentifierProperty: "address",
		IncludeProperties:  []string{"verified", "createdAt"},
	}

	result := BuildPropertyMap("email0", mapping)

	// Should use map projection syntax to avoid implicit grouping expressions
	assert.Equal(t, "email0{.address, .verified, .createdAt}", result)
}

func TestBuildPropertyMap_AllProperties(t *testing.T) {
	mapping := AttributeMapping{
		IdentifierProperty: "number",
		// No IncludeProperties - should use .* projection
	}

	result := BuildPropertyMap("phone0", mapping)

	// Should use map projection with .* to get all properties
	assert.Equal(t, "phone0{.number, .*}", result)
}

func TestBuildPropertyMap_NoIdentifier(t *testing.T) {
	mapping := AttributeMapping{
		IncludeProperties: []string{"street", "city", "state"},
	}

	result := BuildPropertyMap("addr0", mapping)

	// Should use map projection syntax without identifier
	assert.Equal(t, "addr0{.street, .city, .state}", result)
}

func TestBuildPropertyMap_NoIdentifierNoProperties(t *testing.T) {
	mapping := AttributeMapping{
		// No identifier, no include properties - just return everything
	}

	result := BuildPropertyMap("node0", mapping)

	// Should use .* to return all properties
	assert.Equal(t, "node0{.*}", result)
}

func TestSanitizeIdentifier(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"contact_information", "contactinformation"},
		{"identity-documents", "identitydocuments"},
		{"123invalid", "v123invalid"},
		{"valid_name_123", "validname123"},
		{"", "var"},
		{"!!!@@@###", "var"},
		{"email", "email"},
		{"CamelCase", "CamelCase"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := SanitizeIdentifier(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIntegration_CompleteQuery(t *testing.T) {
	// Simulate building a complete query with multiple components
	matchBuilder := NewOptionalMatchBuilder()

	// Add attribute matches
	emailVar := matchBuilder.AddAttributeMatch("c", AttributeMapping{
		RelationshipType: "HAS_EMAIL",
		TargetLabel:      "Email",
	})

	phoneVar := matchBuilder.AddAttributeMatch("c", AttributeMapping{
		RelationshipType: "HAS_PHONE",
		TargetLabel:      "Phone",
	})

	// Build collection for email
	emailColl := NewCollectionBuilder()
	emailColl.AddProperty("address", emailVar, "address")
	emailColl.AddProperty("verified", emailVar, "verified")

	// Build collection for phone
	phoneColl := NewCollectionBuilder()
	phoneColl.AddProperty("number", phoneVar, "number")
	phoneColl.AddProperty("type", phoneVar, "type")

	// Assemble query
	query := strings.Builder{}
	query.WriteString("MATCH (c:Customer {customerId: $customerId})\n")
	query.WriteString(matchBuilder.Build())
	query.WriteString("\nRETURN {\n")
	query.WriteString("  emails: " + emailColl.BuildDistinctCollection() + ",\n")
	query.WriteString("  phones: " + phoneColl.BuildDistinctCollection() + "\n")
	query.WriteString("}")

	result := query.String()

	// Verify complete query structure
	assert.Contains(t, result, "MATCH (c:Customer {customerId: $customerId})")
	assert.Contains(t, result, "OPTIONAL MATCH (c)-[:HAS_EMAIL]->(attr0:Email)")
	assert.Contains(t, result, "OPTIONAL MATCH (c)-[:HAS_PHONE]->(attr1:Phone)")
	// Note: CollectionBuilder still uses old syntax - only BuildPropertyMap uses projection
	assert.Contains(t, result, "emails: collect(DISTINCT {address: attr0.address, verified: attr0.verified})")
	assert.Contains(t, result, "phones: collect(DISTINCT {number: attr1.number, type: attr1.type})")
}
