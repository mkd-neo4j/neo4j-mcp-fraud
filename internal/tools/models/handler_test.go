package models_test

import (
	"context"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	analytics "github.com/mkd-neo4j/neo4j-mcp-fraud/internal/analytics/mocks"
	"github.com/mkd-neo4j/neo4j-mcp-fraud/internal/tools"
	"github.com/mkd-neo4j/neo4j-mcp-fraud/internal/tools/models"
	"go.uber.org/mock/gomock"
)

func TestGetReferenceModelsHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	analyticsService := analytics.NewMockService(ctrl)
	analyticsService.EXPECT().NewToolsEvent(gomock.Any()).AnyTimes()
	analyticsService.EXPECT().EmitEvent(gomock.Any()).AnyTimes()

	t.Run("successfully fetches reference models", func(t *testing.T) {
		deps := &tools.ToolDependencies{
			AnalyticsService: analyticsService,
		}

		handler := models.GetReferenceModelsHandler(deps)
		result, err := handler(context.Background(), mcp.CallToolRequest{})

		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if result == nil || result.IsError {
			t.Error("Expected success result")
		}

		// Verify response contains reference model data
		textContent := result.Content[0].(mcp.TextContent)
		text := textContent.Text

		// Should contain standard fraud detection nodes from Neo4j reference models
		if !strings.Contains(text, "Transaction") && !strings.Contains(text, "Account") && !strings.Contains(text, "Customer") {
			t.Error("Expected reference models to contain standard fraud detection nodes")
		}

		// Should indicate where models came from
		if !strings.Contains(text, "Reference Model from") {
			t.Error("Expected reference models to indicate source URLs")
		}
	})

	t.Run("nil analytics service", func(t *testing.T) {
		deps := &tools.ToolDependencies{
			AnalyticsService: nil,
		}

		handler := models.GetReferenceModelsHandler(deps)
		result, err := handler(context.Background(), mcp.CallToolRequest{})

		if err != nil {
			t.Errorf("Expected no error from handler, got: %v", err)
		}
		if result == nil || !result.IsError {
			t.Error("Expected error result for nil analytics service")
		}
	})
}
