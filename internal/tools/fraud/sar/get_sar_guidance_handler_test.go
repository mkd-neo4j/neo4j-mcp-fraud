package sar_test

import (
	"context"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	analytics "github.com/mkd-neo4j/neo4j-mcp-fraud/internal/analytics/mocks"
	"github.com/mkd-neo4j/neo4j-mcp-fraud/internal/tools"
	"github.com/mkd-neo4j/neo4j-mcp-fraud/internal/tools/fraud/sar"
	"go.uber.org/mock/gomock"
)

func TestGetSARGuidanceHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	analyticsService := analytics.NewMockService(ctrl)
	analyticsService.EXPECT().NewToolsEvent(gomock.Any()).AnyTimes()
	analyticsService.EXPECT().EmitEvent(gomock.Any()).AnyTimes()

	t.Run("successfully returns SAR guidance", func(t *testing.T) {
		deps := &tools.ToolDependencies{
			AnalyticsService: analyticsService,
		}

		handler := sar.GetSARGuidanceHandler(deps)
		result, err := handler(context.Background(), mcp.CallToolRequest{})

		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if result == nil || result.IsError {
			t.Error("Expected success result")
		}

		// Verify response contains SAR guidance
		textContent := result.Content[0].(mcp.TextContent)
		content := textContent.Text

		// Verify key sections are present
		requiredSections := []string{
			"SUSPICIOUS ACTIVITY REPORT (SAR) FILING GUIDANCE",
			"REGULATORY OVERVIEW",
			"FinCEN Forms",
			"Filing Thresholds",
			"Filing Deadlines",
			"REQUIRED SAR COMPONENTS",
			"Part I: Subject Information",
			"Part II: Suspicious Activity Information",
			"Part III: Information About Financial Institution",
			"Part IV: SAR Narrative",
			"NEO4J EVIDENCE GATHERING QUERIES",
			"SUPPORTING DOCUMENTATION REQUIREMENTS",
			"COMPLIANCE BEST PRACTICES",
			"COMMON SAR TYPOLOGIES IN FRAUD DETECTION",
		}

		for _, section := range requiredSections {
			if !strings.Contains(content, section) {
				t.Errorf("Expected content to contain section: %s", section)
			}
		}
	})

	t.Run("contains Neo4j query examples", func(t *testing.T) {
		deps := &tools.ToolDependencies{
			AnalyticsService: analyticsService,
		}

		handler := sar.GetSARGuidanceHandler(deps)
		result, err := handler(context.Background(), mcp.CallToolRequest{})

		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		textContent := result.Content[0].(mcp.TextContent)
		content := textContent.Text

		// Verify Neo4j query examples are present
		queryTypes := []string{
			"Subject Profile and Identity Information",
			"Transaction History and Patterns",
			"Synthetic Identity Detection",
			"Network Analysis",
			"Velocity and Volume Analysis",
			"Geographic Anomalies",
		}

		for _, queryType := range queryTypes {
			if !strings.Contains(content, queryType) {
				t.Errorf("Expected content to contain query type: %s", queryType)
			}
		}

		// Verify actual Cypher query snippets
		if !strings.Contains(content, "MATCH (c:Customer") {
			t.Error("Expected content to contain Cypher MATCH statements")
		}
		if !strings.Contains(content, "customerId") {
			t.Error("Expected content to contain customerId parameter references")
		}
		if !strings.Contains(content, ":TRANSACTION") {
			t.Error("Expected content to contain TRANSACTION relationship references")
		}
	})

	t.Run("contains regulatory details", func(t *testing.T) {
		deps := &tools.ToolDependencies{
			AnalyticsService: analyticsService,
		}

		handler := sar.GetSARGuidanceHandler(deps)
		result, err := handler(context.Background(), mcp.CallToolRequest{})

		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		textContent := result.Content[0].(mcp.TextContent)
		content := textContent.Text

		// Verify regulatory details
		regulatoryDetails := []string{
			"FinCEN Form 111",
			"FinCEN Form 112",
			"$5,000",
			"$2,000",
			"30 calendar days",
			"60 calendar days",
			"Identity theft",
			"Money laundering",
			"Synthetic identity fraud",
		}

		for _, detail := range regulatoryDetails {
			if !strings.Contains(content, detail) {
				t.Errorf("Expected content to contain regulatory detail: %s", detail)
			}
		}
	})

	t.Run("contains best practices", func(t *testing.T) {
		deps := &tools.ToolDependencies{
			AnalyticsService: analyticsService,
		}

		handler := sar.GetSARGuidanceHandler(deps)
		result, err := handler(context.Background(), mcp.CallToolRequest{})

		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		textContent := result.Content[0].(mcp.TextContent)
		content := textContent.Text

		// Verify best practices sections
		if !strings.Contains(content, "DO:") {
			t.Error("Expected content to contain DO best practices")
		}
		if !strings.Contains(content, "DON'T:") {
			t.Error("Expected content to contain DON'T best practices")
		}
		if !strings.Contains(content, "Confidentiality Requirements") {
			t.Error("Expected content to contain confidentiality requirements")
		}
		if !strings.Contains(content, "STRICTLY CONFIDENTIAL") {
			t.Error("Expected content to emphasize confidentiality")
		}
	})

	t.Run("nil analytics service", func(t *testing.T) {
		deps := &tools.ToolDependencies{
			AnalyticsService: nil,
		}

		handler := sar.GetSARGuidanceHandler(deps)
		result, err := handler(context.Background(), mcp.CallToolRequest{})

		if err != nil {
			t.Errorf("Expected no error from handler, got: %v", err)
		}
		if result == nil || !result.IsError {
			t.Error("Expected error result for nil analytics service")
		}
	})
}

func TestGetSARGuidanceSpec(t *testing.T) {
	spec := sar.GetSARGuidanceSpec()

	if spec.Name != "get-sar-report-guidance" {
		t.Errorf("Expected tool name 'get-sar-report-guidance', got: %s", spec.Name)
	}

	if spec.Description == "" {
		t.Error("Expected non-empty description")
	}

	// Verify description contains key phrases
	descriptionPhrases := []string{
		"Suspicious Activity Reports",
		"SAR",
		"FinCEN",
		"Neo4j",
	}

	for _, phrase := range descriptionPhrases {
		if !strings.Contains(spec.Description, phrase) {
			t.Errorf("Expected description to contain phrase: %s", phrase)
		}
	}
}

func TestGetSARGuidanceHandler_ContentStructure(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	analyticsService := analytics.NewMockService(ctrl)
	analyticsService.EXPECT().NewToolsEvent(gomock.Any()).AnyTimes()
	analyticsService.EXPECT().EmitEvent(gomock.Any()).AnyTimes()

	deps := &tools.ToolDependencies{
		AnalyticsService: analyticsService,
	}

	handler := sar.GetSARGuidanceHandler(deps)
	result, err := handler(context.Background(), mcp.CallToolRequest{})

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	textContent := result.Content[0].(mcp.TextContent)
	content := textContent.Text

	// Verify markdown structure
	if !strings.HasPrefix(content, "# SUSPICIOUS ACTIVITY REPORT") {
		t.Error("Expected content to start with main heading")
	}

	// Count major sections (indicated by ##)
	majorSections := strings.Count(content, "\n## ")
	if majorSections < 5 {
		t.Errorf("Expected at least 5 major sections, got: %d", majorSections)
	}

	// Verify presence of query examples (they should be indented code blocks)
	if !strings.Contains(content, "MATCH (c:Customer") {
		t.Error("Expected content to contain Cypher query examples")
	}
}
