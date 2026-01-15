package query_builder

// AttributeMapping defines how to retrieve a specific attribute from the graph.
// This is a schema-agnostic type that can be used for any node-to-node relationship traversal.
// Inspired by PIIRelationship from detect-synthetic-identity but generalized for any use case.
type AttributeMapping struct {
	// RelationshipType is the relationship type name from the schema (e.g., "HAS_EMAIL", "OWNS")
	RelationshipType string `json:"relationshipType"`

	// TargetLabel is the node label of the connected entity (e.g., "Email", "Account")
	TargetLabel string `json:"targetLabel"`

	// IdentifierProperty is the primary property containing the key identifier.
	// Can be empty if all properties should be returned.
	// Examples: "address" for Email, "number" for Phone/SSN, "accountNumber" for Account
	IdentifierProperty string `json:"identifierProperty,omitempty"`

	// AttributeCategory is a logical grouping for organizing output.
	// Examples: "contact_information", "identity_documents", "account_information"
	AttributeCategory string `json:"attributeCategory,omitempty"`

	// IncludeProperties specifies which properties to retrieve from the target node.
	// If empty, all properties are returned using properties() function.
	IncludeProperties []string `json:"includeProperties,omitempty"`
}

// PathSpecification defines a graph traversal path for finding related nodes.
// Used for multi-hop traversals and relationship pattern matching.
type PathSpecification struct {
	// RelationshipType is the relationship type to traverse (e.g., "TRANSACTION", "KNOWS")
	RelationshipType string `json:"relationshipType"`

	// Direction specifies the relationship direction: "out", "in", or "both"
	Direction string `json:"direction"`

	// TargetLabel is the expected node label at the end of the path
	TargetLabel string `json:"targetLabel"`

	// MinHops is the minimum number of hops (relationships) to traverse. 0 means no minimum.
	MinHops int `json:"minHops,omitempty"`

	// MaxHops is the maximum number of hops to traverse. 0 means unlimited (use with caution).
	MaxHops int `json:"maxHops,omitempty"`
}

// PropertyFilter defines filtering criteria for node or relationship properties.
type PropertyFilter struct {
	// PropertyName is the property to filter on
	PropertyName string `json:"propertyName"`

	// Operator defines the comparison operator.
	// Supported: "=", ">", "<", ">=", "<=", "CONTAINS", "STARTS WITH", "ENDS WITH", "IN"
	Operator string `json:"operator"`

	// Value is the value to compare against
	Value interface{} `json:"value"`
}
