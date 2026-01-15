package synthetic_identity_test

import (
	"context"
	"errors"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	analytics "github.com/mkd-neo4j/neo4j-mcp-fraud/internal/analytics/mocks"
	db "github.com/mkd-neo4j/neo4j-mcp-fraud/internal/database/mocks"
	"github.com/mkd-neo4j/neo4j-mcp-fraud/internal/tools"
	"github.com/mkd-neo4j/neo4j-mcp-fraud/internal/tools/fraud/synthetic_identity"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"go.uber.org/mock/gomock"
)

func TestDetectSyntheticIdentityHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	analyticsService := analytics.NewMockService(ctrl)
	analyticsService.EXPECT().NewToolsEvent("detect-synthetic-identity").AnyTimes()
	analyticsService.EXPECT().EmitEvent(gomock.Any()).AnyTimes()
	defer ctrl.Finish()

	t.Run("successful detection with default minSharedAttributes", func(t *testing.T) {
		mockDB := db.NewMockService(ctrl)
		mockDB.EXPECT().
			ExecuteReadQuery(gomock.Any(), gomock.Any(), map[string]any{
				"entityId":            "CUS123",
				"minSharedAttributes": 2,
				"limit":               20,
			}).
			Return([]*neo4j.Record{}, nil)
		mockDB.EXPECT().
			Neo4jRecordsToJSON(gomock.Any()).
			Return(`[{"otherId": "CUS456", "otherFirstName": "Jane", "otherLastName": "Doe", "sharedAttributeCount": 2}]`, nil)

		deps := &tools.ToolDependencies{
			DBService:        mockDB,
			AnalyticsService: analyticsService,
		}

		handler := synthetic_identity.Handler(deps)
		request := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Arguments: map[string]any{
					"entityId": "CUS123",
					"entityConfig": map[string]any{
						"nodeLabel":         "Customer",
						"idProperty":        "customerId",
						"displayProperties": []string{"firstName", "lastName"},
					},
					"piiRelationships": []map[string]any{
						{
							"relationshipType":   "HAS_EMAIL",
							"targetLabel":        "Email",
							"identifierProperty": "address",
						},
						{
							"relationshipType":   "HAS_PHONE",
							"targetLabel":        "Phone",
							"identifierProperty": "number",
						},
					},
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

	t.Run("successful detection with custom minSharedAttributes", func(t *testing.T) {
		mockDB := db.NewMockService(ctrl)
		mockDB.EXPECT().
			ExecuteReadQuery(gomock.Any(), gomock.Any(), map[string]any{
				"entityId":            "CUS123",
				"minSharedAttributes": 3,
				"limit":               20,
			}).
			Return([]*neo4j.Record{}, nil)
		mockDB.EXPECT().
			Neo4jRecordsToJSON(gomock.Any()).
			Return(`[{"otherId": "CUS789", "sharedAttributeCount": 3}]`, nil)

		deps := &tools.ToolDependencies{
			DBService:        mockDB,
			AnalyticsService: analyticsService,
		}

		handler := synthetic_identity.Handler(deps)
		request := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Arguments: map[string]any{
					"entityId":            "CUS123",
					"minSharedAttributes": 3,
					"entityConfig": map[string]any{
						"nodeLabel":         "Customer",
						"idProperty":        "customerId",
						"displayProperties": []string{"firstName", "lastName"},
					},
					"piiRelationships": []map[string]any{
						{
							"relationshipType":   "HAS_EMAIL",
							"targetLabel":        "Email",
							"identifierProperty": "address",
						},
						{
							"relationshipType":   "HAS_PHONE",
							"targetLabel":        "Phone",
							"identifierProperty": "number",
						},
						{
							"relationshipType":   "HAS_PASSPORT",
							"targetLabel":        "Passport",
							"identifierProperty": "passportNumber",
						},
					},
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

	t.Run("discovery mode without entityId", func(t *testing.T) {
		mockDB := db.NewMockService(ctrl)
		mockDB.EXPECT().
			ExecuteReadQuery(gomock.Any(), gomock.Any(), map[string]any{
				"minSharedAttributes": 2,
				"limit":               20,
			}).
			Return([]*neo4j.Record{}, nil)
		mockDB.EXPECT().
			Neo4jRecordsToJSON(gomock.Any()).
			Return(`[{"e1Id": "CUS123", "e2Id": "CUS456", "sharedAttributeCount": 2}]`, nil)

		deps := &tools.ToolDependencies{
			DBService:        mockDB,
			AnalyticsService: analyticsService,
		}

		handler := synthetic_identity.Handler(deps)
		request := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Arguments: map[string]any{
					"entityConfig": map[string]any{
						"nodeLabel":         "Customer",
						"idProperty":        "customerId",
						"displayProperties": []string{"firstName", "lastName"},
					},
					"piiRelationships": []map[string]any{
						{
							"relationshipType":   "HAS_EMAIL",
							"targetLabel":        "Email",
							"identifierProperty": "address",
						},
					},
				},
			},
		}

		result, err := handler(context.Background(), request)

		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if result == nil || result.IsError {
			t.Error("Expected success result for discovery mode")
		}
	})

	t.Run("missing piiRelationships parameter", func(t *testing.T) {
		mockDB := db.NewMockService(ctrl)

		deps := &tools.ToolDependencies{
			DBService:        mockDB,
			AnalyticsService: analyticsService,
		}

		handler := synthetic_identity.Handler(deps)
		request := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Arguments: map[string]any{
					"entityId": "CUS123",
					"entityConfig": map[string]any{
						"nodeLabel":  "Customer",
						"idProperty": "customerId",
					},
					"minSharedAttributes": 2,
				},
			},
		}

		result, err := handler(context.Background(), request)

		if err != nil {
			t.Errorf("Expected no error from handler, got: %v", err)
		}
		if result == nil || !result.IsError {
			t.Error("Expected error result for missing piiRelationships")
		}
	})

	t.Run("invalid arguments binding", func(t *testing.T) {
		mockDB := db.NewMockService(ctrl)

		deps := &tools.ToolDependencies{
			DBService:        mockDB,
			AnalyticsService: analyticsService,
		}

		handler := synthetic_identity.Handler(deps)
		request := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Arguments: "invalid string instead of map",
			},
		}

		result, err := handler(context.Background(), request)

		if err != nil {
			t.Errorf("Expected no error from handler, got: %v", err)
		}
		if result == nil || !result.IsError {
			t.Error("Expected error result for invalid arguments")
		}
	})

	t.Run("nil database service", func(t *testing.T) {
		deps := &tools.ToolDependencies{
			DBService:        nil,
			AnalyticsService: analyticsService,
		}

		handler := synthetic_identity.Handler(deps)
		request := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Arguments: map[string]any{
					"customerId": "CUS123",
				},
			},
		}

		result, err := handler(context.Background(), request)

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

		handler := synthetic_identity.Handler(deps)
		result, err := handler(context.Background(), mcp.CallToolRequest{})

		if err != nil {
			t.Errorf("Expected no error from handler, got: %v", err)
		}
		if result == nil || !result.IsError {
			t.Error("Expected error result for nil analytics service")
		}
	})

	t.Run("database query execution failure", func(t *testing.T) {
		mockDB := db.NewMockService(ctrl)
		mockDB.EXPECT().
			ExecuteReadQuery(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(nil, errors.New("database connection error"))

		deps := &tools.ToolDependencies{
			DBService:        mockDB,
			AnalyticsService: analyticsService,
		}

		handler := synthetic_identity.Handler(deps)
		request := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Arguments: map[string]any{
					"entityId": "CUS123",
					"entityConfig": map[string]any{
						"nodeLabel":  "Customer",
						"idProperty": "customerId",
					},
					"piiRelationships": []map[string]any{
						{
							"relationshipType":   "HAS_EMAIL",
							"targetLabel":        "Email",
							"identifierProperty": "address",
						},
					},
				},
			},
		}

		result, err := handler(context.Background(), request)

		if err != nil {
			t.Errorf("Expected no error from handler, got: %v", err)
		}
		if result == nil || !result.IsError {
			t.Error("Expected error result for query execution failure")
		}
	})

	t.Run("JSON formatting failure", func(t *testing.T) {
		mockDB := db.NewMockService(ctrl)
		mockDB.EXPECT().
			ExecuteReadQuery(gomock.Any(), gomock.Any(), gomock.Any()).
			Return([]*neo4j.Record{}, nil)
		mockDB.EXPECT().
			Neo4jRecordsToJSON(gomock.Any()).
			Return("", errors.New("JSON marshaling failed"))

		deps := &tools.ToolDependencies{
			DBService:        mockDB,
			AnalyticsService: analyticsService,
		}

		handler := synthetic_identity.Handler(deps)
		request := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Arguments: map[string]any{
					"entityId": "CUS123",
					"entityConfig": map[string]any{
						"nodeLabel":  "Customer",
						"idProperty": "customerId",
					},
					"piiRelationships": []map[string]any{
						{
							"relationshipType":   "HAS_EMAIL",
							"targetLabel":        "Email",
							"identifierProperty": "address",
						},
					},
				},
			},
		}

		result, err := handler(context.Background(), request)

		if err != nil {
			t.Errorf("Expected no error from handler, got: %v", err)
		}
		if result == nil || !result.IsError {
			t.Error("Expected error result for JSON formatting failure")
		}
	})
}
