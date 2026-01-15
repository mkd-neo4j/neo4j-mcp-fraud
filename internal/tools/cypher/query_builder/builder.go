package query_builder

import (
	"fmt"
	"strings"
)

// OptionalMatchBuilder helps construct OPTIONAL MATCH clauses dynamically.
// This allows building schema-aware queries without hardcoding relationship names or node labels.
type OptionalMatchBuilder struct {
	clauses    []string
	varCounter int
}

// NewOptionalMatchBuilder creates a new builder instance.
func NewOptionalMatchBuilder() *OptionalMatchBuilder {
	return &OptionalMatchBuilder{
		clauses:    make([]string, 0),
		varCounter: 0,
	}
}

// AddAttributeMatch adds an OPTIONAL MATCH clause for an attribute relationship.
// Returns the generated variable name for use in RETURN clauses.
//
// Example:
//
//	varName := builder.AddAttributeMatch("c", AttributeMapping{
//	    RelationshipType: "HAS_EMAIL",
//	    TargetLabel: "Email",
//	})
//	// Generates: OPTIONAL MATCH (c)-[:HAS_EMAIL]->(attr0:Email)
//	// Returns: "attr0"
func (b *OptionalMatchBuilder) AddAttributeMatch(
	sourceVar string,
	mapping AttributeMapping,
) string {
	varName := fmt.Sprintf("attr%d", b.varCounter)
	b.varCounter++

	clause := fmt.Sprintf("OPTIONAL MATCH (%s)-[:%s]->(%s:%s)",
		sourceVar,
		mapping.RelationshipType,
		varName,
		mapping.TargetLabel)

	b.clauses = append(b.clauses, clause)
	return varName
}

// AddPathMatch adds an OPTIONAL MATCH clause for a path traversal.
// Returns the generated variable name for the end node.
//
// Example:
//
//	varName := builder.AddPathMatch("c", PathSpecification{
//	    RelationshipType: "KNOWS",
//	    Direction: "out",
//	    TargetLabel: "Person",
//	    MinHops: 1,
//	    MaxHops: 3,
//	})
//	// Generates: OPTIONAL MATCH (c)-[:KNOWS*1..3]->(path0:Person)
//	// Returns: "path0"
func (b *OptionalMatchBuilder) AddPathMatch(
	sourceVar string,
	path PathSpecification,
) string {
	varName := fmt.Sprintf("path%d", b.varCounter)
	b.varCounter++

	// Build hop specification
	hopSpec := ""
	if path.MinHops > 0 || path.MaxHops > 0 {
		if path.MinHops == path.MaxHops && path.MinHops > 0 {
			hopSpec = fmt.Sprintf("*%d", path.MinHops)
		} else if path.MaxHops > 0 {
			if path.MinHops > 0 {
				hopSpec = fmt.Sprintf("*%d..%d", path.MinHops, path.MaxHops)
			} else {
				hopSpec = fmt.Sprintf("*..%d", path.MaxHops)
			}
		} else if path.MinHops > 0 {
			hopSpec = fmt.Sprintf("*%d..", path.MinHops)
		}
	}

	// Build relationship pattern based on direction
	var clause string
	if path.Direction == "in" {
		clause = fmt.Sprintf("OPTIONAL MATCH (%s)<-[:%s%s]-(%s:%s)",
			sourceVar,
			path.RelationshipType,
			hopSpec,
			varName,
			path.TargetLabel)
	} else if path.Direction == "both" {
		clause = fmt.Sprintf("OPTIONAL MATCH (%s)-[:%s%s]-(%s:%s)",
			sourceVar,
			path.RelationshipType,
			hopSpec,
			varName,
			path.TargetLabel)
	} else {
		// Default to "out"
		clause = fmt.Sprintf("OPTIONAL MATCH (%s)-[:%s%s]->(%s:%s)",
			sourceVar,
			path.RelationshipType,
			hopSpec,
			varName,
			path.TargetLabel)
	}

	b.clauses = append(b.clauses, clause)
	return varName
}

// AddCustomMatch adds a custom OPTIONAL MATCH clause.
// Use this for complex patterns not covered by the helper methods.
func (b *OptionalMatchBuilder) AddCustomMatch(clause string) {
	b.clauses = append(b.clauses, "OPTIONAL MATCH "+clause)
}

// Build returns all OPTIONAL MATCH clauses as a single string.
func (b *OptionalMatchBuilder) Build() string {
	if len(b.clauses) == 0 {
		return ""
	}
	return strings.Join(b.clauses, "\n")
}

// GetClauseCount returns the number of OPTIONAL MATCH clauses added.
func (b *OptionalMatchBuilder) GetClauseCount() int {
	return len(b.clauses)
}

// CollectionBuilder helps construct collect(DISTINCT {...}) expressions for RETURN clauses.
type CollectionBuilder struct {
	items []string
}

// NewCollectionBuilder creates a new collection builder.
func NewCollectionBuilder() *CollectionBuilder {
	return &CollectionBuilder{
		items: make([]string, 0),
	}
}

// AddProperty adds a single property to the collection map.
//
// Example:
//
//	builder.AddProperty("email", "e", "address")
//	// Generates: email: e.address
func (c *CollectionBuilder) AddProperty(propName string, sourceVar string, sourceProp string) {
	c.items = append(c.items, fmt.Sprintf("%s: %s.%s", propName, sourceVar, sourceProp))
}

// AddAllProperties adds all properties from a node using the properties() function.
//
// Example:
//
//	builder.AddAllProperties("email", "e")
//	// Generates: email: properties(e)
func (c *CollectionBuilder) AddAllProperties(key string, sourceVar string) {
	c.items = append(c.items, fmt.Sprintf("%s: properties(%s)", key, sourceVar))
}

// AddCustomExpression adds a custom key-value expression.
//
// Example:
//
//	builder.AddCustomExpression("fullName", "c.firstName + ' ' + c.lastName")
//	// Generates: fullName: c.firstName + ' ' + c.lastName
func (c *CollectionBuilder) AddCustomExpression(key string, expression string) {
	c.items = append(c.items, fmt.Sprintf("%s: %s", key, expression))
}

// Build returns the collection as a map expression.
//
// Example: {email: e.address, verified: e.verified}
func (c *CollectionBuilder) Build() string {
	if len(c.items) == 0 {
		return "{}"
	}
	return "{" + strings.Join(c.items, ", ") + "}"
}

// BuildDistinctCollection wraps the map in collect(DISTINCT {...}).
//
// Example: collect(DISTINCT {email: e.address, verified: e.verified})
func (c *CollectionBuilder) BuildDistinctCollection() string {
	return "collect(DISTINCT " + c.Build() + ")"
}

// BuildCollection wraps the map in collect({...}) without DISTINCT.
func (c *CollectionBuilder) BuildCollection() string {
	return "collect(" + c.Build() + ")"
}

// GroupMappingsByCategory organizes attribute mappings by their category.
// Returns a map of category -> []AttributeMapping.
//
// This is useful for generating organized output grouped by logical sections
// (e.g., contact_information, identity_documents, account_information).
func GroupMappingsByCategory(mappings []AttributeMapping) map[string][]AttributeMapping {
	categorized := make(map[string][]AttributeMapping)

	for _, mapping := range mappings {
		category := mapping.AttributeCategory
		if category == "" {
			category = "other_attributes"
		}
		categorized[category] = append(categorized[category], mapping)
	}

	return categorized
}

// BuildPropertyMap constructs a map projection expression for a single attribute mapping.
// Uses Neo4j map projection syntax to avoid implicit grouping expression errors in aggregations.
//
// Example:
//
//	expr := BuildPropertyMap("email0", AttributeMapping{
//	    IdentifierProperty: "address",
//	    IncludeProperties: []string{"verified", "createdAt"},
//	})
//	// Returns: email0{.address, .verified, .createdAt}
//
// For all properties:
//
//	expr := BuildPropertyMap("email0", AttributeMapping{
//	    IdentifierProperty: "address",
//	})
//	// Returns: email0{.address, .*}
func BuildPropertyMap(varName string, mapping AttributeMapping) string {
	var projections []string

	if len(mapping.IncludeProperties) > 0 {
		// Include specific properties using map projection syntax
		if mapping.IdentifierProperty != "" {
			projections = append(projections, "."+mapping.IdentifierProperty)
		}
		for _, prop := range mapping.IncludeProperties {
			projections = append(projections, "."+prop)
		}
		return fmt.Sprintf("%s{%s}", varName, strings.Join(projections, ", "))
	}

	// Return all properties using .* projection
	if mapping.IdentifierProperty != "" {
		// Include identifier explicitly, then all other properties
		return fmt.Sprintf("%s{.%s, .*}", varName, mapping.IdentifierProperty)
	}

	// Just return all properties
	return fmt.Sprintf("%s{.*}", varName)
}

// SanitizeIdentifier sanitizes a string to be used as a Cypher variable name.
// Removes special characters and ensures valid identifier format.
func SanitizeIdentifier(s string) string {
	// Replace non-alphanumeric characters with empty string
	var result strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			result.WriteRune(r)
		}
	}

	sanitized := result.String()

	// Ensure it starts with a letter
	if len(sanitized) > 0 && sanitized[0] >= '0' && sanitized[0] <= '9' {
		sanitized = "v" + sanitized
	}

	// Ensure it's not empty
	if sanitized == "" {
		sanitized = "var"
	}

	return sanitized
}
