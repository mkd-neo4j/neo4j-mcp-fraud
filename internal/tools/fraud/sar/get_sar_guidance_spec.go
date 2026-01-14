package sar

import "github.com/mark3labs/mcp-go/mcp"

// GetSARGuidanceSpec returns the tool specification for get-sar-report-guidance
func GetSARGuidanceSpec() mcp.Tool {
	return mcp.NewTool("get-sar-report-guidance",
		mcp.WithDescription(`
		Fetches comprehensive guidance on creating Suspicious Activity Reports (SARs) for financial institutions.

		Returns structured guidance including:
		- Required information and FinCEN form requirements (Forms 111/112)
		- Filing thresholds and regulatory timelines
		- Key components of a complete SAR filing
		- Neo4j Cypher query patterns for gathering evidence from the fraud detection graph
		- Supporting documentation requirements
		- Best practices for SAR narrative construction

		This tool provides reference documentation that can be used to:
		- Understand what information is required to file a SAR
		- Learn the regulatory thresholds and deadlines
		- Discover how to query Neo4j to gather supporting evidence
		- Ensure compliance with FinCEN BSA reporting requirements

		Use this tool when you need guidance on:
		- How to structure and create a SAR report
		- What data points are required for SAR filing
		- Which Neo4j queries to run to gather SAR evidence
		- Regulatory requirements and timelines for SAR filing
		- Building SAR narratives based on fraud detection findings`),
		mcp.WithTitleAnnotation("Get SAR Report Guidance"),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithOpenWorldHintAnnotation(true),
	)
}
