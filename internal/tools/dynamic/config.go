package dynamic

// ToolConfig represents the YAML configuration for a dynamic tool
type ToolConfig struct {
	// Name is the unique tool identifier (e.g., "detect-synthetic-identity")
	Name string `yaml:"name"`

	// Description provides the operational description of the tool
	Description string `yaml:"description"`

	// Intent provides semantic understanding for agents - WHEN to use this tool
	Intent string `yaml:"intent,omitempty"`

	// ExpectedPatterns describes the patterns this tool helps detect
	ExpectedPatterns []PatternConfig `yaml:"expected_patterns,omitempty"`

	// ReferenceCypher provides canonical query implementation as guidance for the LLM
	ReferenceCypher string `yaml:"reference_cypher,omitempty"`

	// ReferenceSchema provides hints about common labels/relationships to look for
	ReferenceSchema *ReferenceSchemaConfig `yaml:"reference_schema,omitempty"`

	// Parameters defines typed input parameters for the query
	Parameters []ParameterConfig `yaml:"parameters,omitempty"`

	// Category is derived from the folder structure (e.g., "fraud", "graph-data")
	// This is an internal field, not from YAML
	Category string `yaml:"-"`
}

// PatternConfig describes an expected detection pattern
type PatternConfig struct {
	// Entity is the node type being analyzed (e.g., "Application", "Person")
	Entity string `yaml:"entity"`

	// SharedElements are the PII or attributes that may be shared
	SharedElements []string `yaml:"shared_elements,omitempty"`

	// Anomaly describes what makes this pattern suspicious
	Anomaly string `yaml:"anomaly"`
}

// ReferenceSchemaConfig provides hints about common graph elements
type ReferenceSchemaConfig struct {
	// Labels are common node labels to look for
	Labels []string `yaml:"labels,omitempty"`

	// Relationships are common relationship types to look for
	Relationships []string `yaml:"relationships,omitempty"`
}

// ParameterConfig defines a typed input parameter
type ParameterConfig struct {
	// Name is the parameter identifier
	Name string `yaml:"name"`

	// Type is the JSON Schema type (string, integer, number, boolean, array, object)
	Type string `yaml:"type"`

	// Description explains the parameter's purpose
	Description string `yaml:"description,omitempty"`

	// Default value (type depends on Type field)
	Default interface{} `yaml:"default,omitempty"`

	// Required indicates if this parameter must be provided
	Required bool `yaml:"required,omitempty"`
}
