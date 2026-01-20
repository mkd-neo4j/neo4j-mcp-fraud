package dynamic

// ToolConfig represents the YAML configuration for a dynamic tool
type ToolConfig struct {
	// Name is the unique tool identifier (e.g., "detect-synthetic-identity")
	Name string `yaml:"name"`

	// Title is the human-readable tool name
	Title string `yaml:"title"`

	// Description provides detailed guidance for LLMs on how to use this tool
	Description string `yaml:"description"`

	// InputSchema defines the JSON schema for tool inputs (optional for documentation tools)
	InputSchema *InputSchemaConfig `yaml:"input_schema,omitempty"`

	// Execution defines how the query should be executed (optional for documentation tools)
	Execution *ExecutionConfig `yaml:"execution,omitempty"`

	// Metadata contains tool categorization and behavior flags
	Metadata MetadataConfig `yaml:"metadata"`

	// StaticContent is the hardcoded content to return (for documentation tools)
	StaticContent string `yaml:"static_content,omitempty"`
}

// InputSchemaConfig defines the JSON schema structure for tool inputs
type InputSchemaConfig struct {
	Type       string                 `yaml:"type"`
	Required   []string               `yaml:"required,omitempty"`
	Properties map[string]PropertyDef `yaml:"properties"`
}

// PropertyDef defines a single property in the input schema
type PropertyDef struct {
	Type        string                 `yaml:"type"`
	Description string                 `yaml:"description"`
	Default     interface{}            `yaml:"default,omitempty"`
	Properties  map[string]PropertyDef `yaml:"properties,omitempty"`
}

// ExecutionConfig defines how the query should be executed
type ExecutionConfig struct {
	// Mode is either "read" or "write"
	Mode string `yaml:"mode"`

	// Timeout in milliseconds
	Timeout int `yaml:"timeout"`
}

// MetadataConfig contains tool categorization and behavior flags
type MetadataConfig struct {
	// ReadOnly indicates if this tool only reads data
	ReadOnly bool `yaml:"readonly"`

	// Idempotent indicates if repeated calls produce the same result
	Idempotent bool `yaml:"idempotent"`

	// Destructive indicates if this tool modifies data
	Destructive bool `yaml:"destructive"`

	// Category is derived from the folder structure (e.g., "fraud", "data")
	Category string `yaml:"category"`
}
