package models

import "github.com/mark3labs/mcp-go/mcp"

// GetReferenceModelsSpec returns the tool specification for get-data-models
func GetReferenceModelsSpec() mcp.Tool {
	return mcp.NewTool("get-data-models",
		mcp.WithDescription(`
		Fetches Neo4j official reference data models for fraud detection and banking applications.

		Returns the standard Neo4j data model patterns including:
		- Recommended node labels and properties for fraud detection
		- Standard relationship patterns for banking transactions
		- Best practices for modeling customer identities, accounts, and transactions
		- Fraud investigation patterns (Cases, Alerts, etc.)

		This tool provides reference documentation that can be used to:
		- Understand Neo4j recommended patterns for fraud detection
		- Get guidance on extending an existing schema
		- Learn what properties and relationships are commonly used
		- Compare current implementation against industry standards

		The reference models are independent of your database - they show Neo4j's
		recommended patterns, not what currently exists in your database.

		Use this tool when you need guidance on:
		- How to extend an existing fraud detection schema
		- What properties and relationships are recommended for fraud detection
		- Neo4j best practices for banking and financial crime applications
		- Understanding standard patterns for customer identity, transactions, and accounts`),
		mcp.WithTitleAnnotation("Get Data Models"),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithOpenWorldHintAnnotation(true),
	)
}
