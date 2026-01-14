package schema_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	analytics "github.com/mkd-neo4j/neo4j-mcp-fraud/internal/analytics/mocks"
	db "github.com/mkd-neo4j/neo4j-mcp-fraud/internal/database/mocks"
	"github.com/mkd-neo4j/neo4j-mcp-fraud/internal/tools"
	"github.com/mkd-neo4j/neo4j-mcp-fraud/internal/tools/schema"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"go.uber.org/mock/gomock"
)

func TestEnrichSchemaHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	analyticsService := analytics.NewMockService(ctrl)
	analyticsService.EXPECT().NewToolsEvent(gomock.Any()).AnyTimes()
	analyticsService.EXPECT().EmitEvent(gomock.Any()).AnyTimes()
	defer ctrl.Finish()

	t.Run("successful schema enrichment with default URLs", func(t *testing.T) {
		mockDB := db.NewMockService(ctrl)
		mockDB.EXPECT().
			ExecuteReadQuery(gomock.Any(), gomock.Any(), gomock.Any()).
			Return([]*neo4j.Record{
				{
					Keys: []string{"key", "value"},
					Values: []any{
						"Customer",
						map[string]any{
							"type": "node",
							"properties": map[string]any{
								"customerId": map[string]any{"type": "STRING"},
								"firstName":  map[string]any{"type": "STRING"},
							},
							"relationships": map[string]any{},
						},
					},
				},
			}, nil)

		deps := &tools.ToolDependencies{
			DBService:        mockDB,
			AnalyticsService: analyticsService,
		}

		handler := schema.EnrichSchemaHandler(deps, 100)
		result, err := handler(context.Background(), mcp.CallToolRequest{})

		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if result == nil || result.IsError {
			t.Error("Expected success result")
		}

		// Verify response structure
		textContent := result.Content[0].(mcp.TextContent)
		var enrichmentReq map[string]interface{}
		if err := json.Unmarshal([]byte(textContent.Text), &enrichmentReq); err != nil {
			t.Errorf("Failed to parse enrichment response: %v", err)
		}

		// Check that required fields are present
		if _, ok := enrichmentReq["raw_schema"]; !ok {
			t.Error("Missing raw_schema field in response")
		}
		if _, ok := enrichmentReq["reference_model"]; !ok {
			t.Error("Missing reference_model field in response")
		}
		if _, ok := enrichmentReq["prompt"]; !ok {
			t.Error("Missing prompt field in response")
		}
		if _, ok := enrichmentReq["instructions"]; !ok {
			t.Error("Missing instructions field in response")
		}
	})

	t.Run("enrichment with custom URL parameter", func(t *testing.T) {
		mockDB := db.NewMockService(ctrl)
		mockDB.EXPECT().
			ExecuteReadQuery(gomock.Any(), gomock.Any(), gomock.Any()).
			Return([]*neo4j.Record{
				{
					Keys: []string{"key", "value"},
					Values: []any{
						"Account",
						map[string]any{
							"type":          "node",
							"properties":    map[string]any{"accountNumber": map[string]any{"type": "STRING"}},
							"relationships": map[string]any{},
						},
					},
				},
			}, nil)

		deps := &tools.ToolDependencies{
			DBService:        mockDB,
			AnalyticsService: analyticsService,
		}

		handler := schema.EnrichSchemaHandler(deps, 100)
		request := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Arguments: map[string]interface{}{
					"reference_model_urls": "https://example.com/model1.txt,https://example.com/model2.txt",
				},
			},
		}
		result, err := handler(context.Background(), request)

		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if result == nil || result.IsError {
			t.Error("Expected success result")
		}
	})

	t.Run("database query failure", func(t *testing.T) {
		mockDB := db.NewMockService(ctrl)
		mockDB.EXPECT().
			ExecuteReadQuery(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(nil, errors.New("connection failed"))

		deps := &tools.ToolDependencies{
			DBService:        mockDB,
			AnalyticsService: analyticsService,
		}

		handler := schema.EnrichSchemaHandler(deps, 100)
		result, err := handler(context.Background(), mcp.CallToolRequest{})

		if err != nil {
			t.Errorf("Expected no error from handler, got: %v", err)
		}
		if result == nil || !result.IsError {
			t.Error("Expected error result")
		}
	})

	t.Run("nil database service", func(t *testing.T) {
		deps := &tools.ToolDependencies{
			DBService:        nil,
			AnalyticsService: analyticsService,
		}

		handler := schema.EnrichSchemaHandler(deps, 100)
		result, err := handler(context.Background(), mcp.CallToolRequest{})

		if err != nil {
			t.Errorf("Expected no error from handler, got: %v", err)
		}
		if result == nil || !result.IsError {
			t.Error("Expected error result for nil database service")
		}
	})

	t.Run("nil analytics service", func(t *testing.T) {
		mockDB := db.NewMockService(ctrl)
		deps := &tools.ToolDependencies{
			DBService:        mockDB,
			AnalyticsService: nil,
		}

		handler := schema.EnrichSchemaHandler(deps, 100)
		result, err := handler(context.Background(), mcp.CallToolRequest{})

		if err != nil {
			t.Errorf("Expected no error from handler, got: %v", err)
		}
		if result == nil || !result.IsError {
			t.Error("Expected error result for nil analytics service")
		}
	})

	t.Run("empty database schema", func(t *testing.T) {
		mockDB := db.NewMockService(ctrl)
		mockDB.EXPECT().
			ExecuteReadQuery(gomock.Any(), gomock.Any(), gomock.Any()).
			Return([]*neo4j.Record{}, nil)

		deps := &tools.ToolDependencies{
			DBService:        mockDB,
			AnalyticsService: analyticsService,
		}

		handler := schema.EnrichSchemaHandler(deps, 100)
		result, err := handler(context.Background(), mcp.CallToolRequest{})

		if err != nil {
			t.Errorf("Expected no error from handler, got: %v", err)
		}

		// Empty schema should still return success with a message
		if result == nil {
			t.Error("Expected non-nil result")
			return
		}

		if result.IsError {
			t.Error("Expected success result for empty database")
		}
	})
}
