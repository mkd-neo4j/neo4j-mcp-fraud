package cypher_test

import (
	"context"
	// "encoding/json" // Commented out - only used in TestGetSchemaProcessing which is now commented out
	"errors"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	analytics "github.com/mkd-neo4j/neo4j-mcp-fraud/internal/analytics/mocks"
	db "github.com/mkd-neo4j/neo4j-mcp-fraud/internal/database/mocks"
	"github.com/mkd-neo4j/neo4j-mcp-fraud/internal/tools"
	"github.com/mkd-neo4j/neo4j-mcp-fraud/internal/tools/cypher"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j/dbtype"
	"go.uber.org/mock/gomock"
)

func TestGetSchemaHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	analyticsService := analytics.NewMockService(ctrl)
	analyticsService.EXPECT().NewToolsEvent("get-schema").AnyTimes()
	analyticsService.EXPECT().EmitEvent(gomock.Any()).AnyTimes()
	defer ctrl.Finish()

	t.Run("successful schema retrieval", func(t *testing.T) {
		mockDB := db.NewMockService(ctrl)

		// Mock GetDatabaseName for logging
		mockDB.EXPECT().
			GetDatabaseName().
			Return("neo4j").
			AnyTimes()

		// Mock db.schema.visualization query
		mockDB.EXPECT().
			ExecuteReadQuery(gomock.Any(), gomock.Eq("CALL db.schema.visualization()"), nil).
			Return([]*neo4j.Record{
				{
					Keys: []string{"nodes", "relationships"},
					Values: []any{
						[]any{
							map[string]any{"name": "Movie"},
							map[string]any{"name": "Person"},
						},
						[]any{
							[]any{
								map[string]any{"name": "Person"},
								"ACTED_IN",
								map[string]any{"name": "Movie"},
							},
						},
					},
				},
			}, nil)

		// Mock db.schema.nodeTypeProperties query
		mockDB.EXPECT().
			ExecuteReadQuery(gomock.Any(), gomock.Any(), nil).
			Return([]*neo4j.Record{
				{
					Keys:   []string{"nodeLabels", "propertyName", "propertyTypes"},
					Values: []any{[]any{"Movie"}, "title", []any{"STRING"}},
				},
			}, nil)

		// Mock db.schema.relTypeProperties query
		mockDB.EXPECT().
			ExecuteReadQuery(gomock.Any(), gomock.Any(), nil).
			Return([]*neo4j.Record{}, nil)

		deps := &tools.ToolDependencies{
			DBService:        mockDB,
			AnalyticsService: analyticsService,
		}

		handler := cypher.GetSchemaHandler(deps, 100)
		result, err := handler(context.Background(), mcp.CallToolRequest{})

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
			GetDatabaseName().
			Return("neo4j").
			AnyTimes()
		mockDB.EXPECT().
			ExecuteReadQuery(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(nil, errors.New("connection failed"))

		deps := &tools.ToolDependencies{
			DBService:        mockDB,
			AnalyticsService: analyticsService,
		}

		handler := cypher.GetSchemaHandler(deps, 100)
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

		handler := cypher.GetSchemaHandler(deps, 100)
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

		handler := cypher.GetSchemaHandler(deps, 100)
		result, err := handler(context.Background(), mcp.CallToolRequest{})

		if err != nil {
			t.Errorf("Expected no error from handler, got: %v", err)
		}
		if result == nil || !result.IsError {
			t.Error("Expected error result for nil analytics service")
		}
	})
	t.Run("No records returned from apoc query (empty database)", func(t *testing.T) {
		analyticsService := analytics.NewMockService(ctrl)
		analyticsService.EXPECT().NewToolsEvent("get-schema").Times(1)
		analyticsService.EXPECT().EmitEvent(gomock.Any()).Times(1)
		mockDB := db.NewMockService(ctrl)
		mockDB.EXPECT().
			GetDatabaseName().
			Return("neo4j").
			AnyTimes()
		// Mock schema visualization returning empty
		mockDB.EXPECT().
			ExecuteReadQuery(gomock.Any(), gomock.Eq("CALL db.schema.visualization()"), nil).
			Return([]*neo4j.Record{}, nil)
		// Mock node count query returning 0
		mockDB.EXPECT().
			ExecuteReadQuery(gomock.Any(), gomock.Eq("MATCH (n) RETURN count(n) as nodeCount"), nil).
			Return([]*neo4j.Record{
				{
					Keys:   []string{"nodeCount"},
					Values: []any{int64(0)},
				},
			}, nil)

		deps := &tools.ToolDependencies{
			DBService:        mockDB,
			AnalyticsService: analyticsService,
		}

		handler := cypher.GetSchemaHandler(deps, 100)
		result, err := handler(context.Background(), mcp.CallToolRequest{})

		if err != nil {
			t.Errorf("Expected no error from handler, got: %v", err)
		}

		if result == nil {
			t.Error("Expected non-nil result")
			return
		}

		if result.IsError {
			t.Error("Expected success result, not error")
			return
		}

		textContent := result.Content[0].(mcp.TextContent)
		if textContent.Text != "The get-schema tool executed successfully; however, since the Neo4j database 'neo4j' contains no data, no schema information was returned." {
			t.Errorf("Expected updated empty database message, got: %s", textContent.Text)
		}
	})

	t.Run("verify proper Cypher syntax in output", func(t *testing.T) {
		mockDB := db.NewMockService(ctrl)

		// Mock GetDatabaseName for logging
		mockDB.EXPECT().
			GetDatabaseName().
			Return("neo4j").
			AnyTimes()

		// Create proper dbtype.Node instances with real IDs
		// NOTE: The "name" property is required for schema visualization
		customerNode := dbtype.Node{
			Id:         1,
			Labels:     []string{"Customer"},
			Props:      map[string]any{"name": "Customer"},
			ElementId:  "4:1",
		}
		passportNode := dbtype.Node{
			Id:         2,
			Labels:     []string{"Passport"},
			Props:      map[string]any{"name": "Passport"},
			ElementId:  "4:2",
		}
		emailNode := dbtype.Node{
			Id:         3,
			Labels:     []string{"Email"},
			Props:      map[string]any{"name": "Email"},
			ElementId:  "4:3",
		}

		// Mock db.schema.visualization query with proper relationships
		mockDB.EXPECT().
			ExecuteReadQuery(gomock.Any(), gomock.Eq("CALL db.schema.visualization()"), nil).
			Return([]*neo4j.Record{
				{
					Keys: []string{"nodes", "relationships"},
					Values: []any{
						[]any{customerNode, passportNode, emailNode},
						[]any{
							dbtype.Relationship{
								Id:        1,
								StartId:   1,
								EndId:     2,
								Type:      "HAS_PASSPORT",
								Props:     map[string]any{"name": "HAS_PASSPORT"},
								ElementId: "5:1",
							},
							dbtype.Relationship{
								Id:        2,
								StartId:   1,
								EndId:     3,
								Type:      "HAS_EMAIL",
								Props:     map[string]any{"name": "HAS_EMAIL"},
								ElementId: "5:2",
							},
						},
					},
				},
			}, nil)

		// Mock db.schema.nodeTypeProperties query
		mockDB.EXPECT().
			ExecuteReadQuery(gomock.Any(), gomock.Any(), nil).
			Return([]*neo4j.Record{
				{
					Keys:   []string{"nodeLabels", "propertyName", "propertyTypes"},
					Values: []any{[]any{"Customer"}, "customerId", []any{"STRING"}},
				},
				{
					Keys:   []string{"nodeLabels", "propertyName", "propertyTypes"},
					Values: []any{[]any{"Passport"}, "number", []any{"STRING"}},
				},
				{
					Keys:   []string{"nodeLabels", "propertyName", "propertyTypes"},
					Values: []any{[]any{"Email"}, "address", []any{"STRING"}},
				},
			}, nil)

		// Mock db.schema.relTypeProperties query
		mockDB.EXPECT().
			ExecuteReadQuery(gomock.Any(), gomock.Any(), nil).
			Return([]*neo4j.Record{}, nil)

		deps := &tools.ToolDependencies{
			DBService:        mockDB,
			AnalyticsService: analyticsService,
		}

		handler := cypher.GetSchemaHandler(deps, 100)
		result, err := handler(context.Background(), mcp.CallToolRequest{})

		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		if result == nil || result.IsError {
			t.Fatal("Expected success result")
		}

		textContent := result.Content[0].(mcp.TextContent)
		output := textContent.Text

		// Verify that the output contains proper Cypher patterns
		expectedPatterns := []string{
			"(:Customer)-[:HAS_PASSPORT]->(:Passport)",
			"(:Customer)-[:HAS_EMAIL]->(:Email)",
		}

		for _, pattern := range expectedPatterns {
			if !strings.Contains(output, pattern) {
				t.Errorf("Expected output to contain Cypher pattern %q, but it was not found.\nOutput:\n%s", pattern, output)
			}
		}

		// Verify that old format (with arrows only) is NOT present
		oldFormatPatterns := []string{
			":HAS_PASSPORT` → ",
			":HAS_EMAIL` → ",
		}

		for _, pattern := range oldFormatPatterns {
			if strings.Contains(output, pattern) {
				t.Errorf("Output should NOT contain old format pattern %q, but it was found.\nOutput:\n%s", pattern, output)
			}
		}

		t.Logf("Schema output:\n%s", output)
	})

}

// TestGetSchemaProcessing tests are commented out because they test the old APOC-based
// processCypherSchema function which is no longer used by the handler (replaced with native Neo4j procedures).
// The processCypherSchema function is kept for potential backward compatibility but is not actively used.
/*
func TestGetSchemaProcessing(t *testing.T) {
	ctrl := gomock.NewController(t)
	analyticsService := analytics.NewMockService(ctrl)
	analyticsService.EXPECT().NewToolsEvent("get-schema").AnyTimes()
	analyticsService.EXPECT().EmitEvent(gomock.Any()).AnyTimes()
	defer ctrl.Finish()

	testCases := []struct {
		name         string
		expectedErr  bool
		mockRecords  []*neo4j.Record
		expectedJSON string
	}{
		{
			name:        "successful schema processing",
			expectedErr: false,
			mockRecords: []*neo4j.Record{
				{
					Keys: []string{"key", "value"},
					Values: []any{
						"Movie",
						map[string]any{
							"type": "node",
							"properties": map[string]any{
								"title":    map[string]any{"type": "STRING", "indexed": false},
								"released": map[string]any{"type": "INTEGER", "indexed": true},
							},
							"relationships": map[string]any{
								"ACTED_IN": map[string]any{
									"count":     16,
									"direction": "in",
									"labels":    []any{"Person"},
									"properties": map[string]any{
										"year": map[string]any{"type": "DATE", "indexed": false},
									},
								},
							},
						},
					},
				},
				{
					Keys: []string{"key", "value"},
					Values: []any{
						"ACTED_IN",
						map[string]any{
							"type": "relationship",
							"properties": map[string]any{
								"roles": map[string]any{"type": "LIST"},
							},
						},
					},
				},
			},
			expectedJSON: `[
				{
					"key": "Movie",
					"value": {
						"properties": {
							"released": "INTEGER",
							"title": "STRING"
						},
						"relationships": {
							"ACTED_IN": {
								"direction": "in",
								"labels": [
									"Person"
								],
								"properties": {
									"year": "DATE"
								}
							}
						},
						"type": "node"
					}
				},
				{
					"key": "ACTED_IN",
					"value": {
						"properties": {
							"roles": "LIST"
						},
						"type": "relationship"
					}
				}
			]`,
		},
		{
			name:        "schema with multiple nodes and varied relationships",
			expectedErr: false,
			mockRecords: []*neo4j.Record{
				{
					Keys: []string{"key", "value"},
					Values: []any{
						"Movie",
						map[string]any{
							"type": "node",
							"properties": map[string]any{
								"title":    map[string]any{"type": "STRING", "indexed": false},
								"released": map[string]any{"type": "INTEGER", "indexed": false},
							},
							"relationships": map[string]any{
								"ACTED_IN": map[string]any{
									"direction": "in", "labels": []any{"Person"}, "properties": map[string]any{"roles": map[string]any{"type": "LIST"}},
								},
								"DIRECTED": map[string]any{
									"direction": "in", "labels": []any{"Person"}, "properties": map[string]any{},
								},
							},
						},
					},
				},
				{
					Keys: []string{"key", "value"},
					Values: []any{
						"Person",
						map[string]any{
							"type":       "node",
							"properties": map[string]any{"name": map[string]any{"type": "STRING"}, "born": map[string]any{"type": "INTEGER"}},
							"relationships": map[string]any{
								"ACTED_IN": map[string]any{
									"direction": "out", "labels": []any{"Movie"}, "properties": map[string]any{"roles": map[string]any{"type": "LIST"}},
								},
								"DIRECTED": map[string]any{
									"direction": "out", "labels": []any{"Movie"}, "properties": map[string]any{},
								},
							},
						},
					},
				},
				{
					Keys:   []string{"key", "value"},
					Values: []any{"ACTED_IN", map[string]any{"type": "relationship", "properties": map[string]any{"roles": map[string]any{"type": "LIST"}}}},
				},
				{
					Keys:   []string{"key", "value"},
					Values: []any{"DIRECTED", map[string]any{"type": "relationship", "properties": map[string]any{}}},
				},
			},
			expectedJSON: `[
				{
					"key": "Movie",
					"value": {
						"properties": {"released": "INTEGER", "title": "STRING"},
						"relationships": {
							"ACTED_IN": {"direction": "in", "labels": ["Person"], "properties": {"roles": "LIST"}},
							"DIRECTED": {"direction": "in", "labels": ["Person"]}
						},
						"type": "node"
					}
				},
				{
					"key": "Person",
					"value": {
						"properties": {"born": "INTEGER", "name": "STRING"},
						"relationships": {
							"ACTED_IN": {"direction": "out", "labels": ["Movie"], "properties": {"roles": "LIST"}},
							"DIRECTED": {"direction": "out", "labels": ["Movie"]}
						},
						"type": "node"
					}
				},
				{
					"key": "ACTED_IN",
					"value": {"properties": {"roles": "LIST"}, "type": "relationship"}
				},
				{
					"key": "DIRECTED",
					"value": {"type": "relationship"}
				}
			]`,
		},
		{
			name:        "schema with a node with no relationships",
			expectedErr: false,
			mockRecords: []*neo4j.Record{
				{
					Keys: []string{"key", "value"},
					Values: []any{
						"Genre",
						map[string]any{
							"type":          "node",
							"properties":    map[string]any{"name": map[string]any{"type": "STRING"}},
							"relationships": map[string]any{},
						},
					},
				},
			},
			expectedJSON: `[
				{
					"key": "Genre",
					"value": {
						"properties": {"name": "STRING"},
						"type": "node"
					}
				}
			]`,
		},
		{
			name:        "schema with a node with no relationships (relationships nil)",
			expectedErr: false,
			mockRecords: []*neo4j.Record{
				{
					Keys: []string{"key", "value"},
					Values: []any{
						"Genre",
						map[string]any{
							"type":          "node",
							"properties":    map[string]any{"name": map[string]any{"type": "STRING"}},
							"relationships": nil,
						},
					},
				},
			},
			expectedJSON: `[
				{
					"key": "Genre",
					"value": {
						"properties": {"name": "STRING"},
						"type": "node"
					}
				}
			]`,
		},
		{
			name:        "should fail for invalid output returned by apoc.meta.schema (no key returned)",
			expectedErr: true,
			mockRecords: []*neo4j.Record{
				{
					Keys: []string{"value"},
					Values: []any{
						"Genre",
					},
				},
			},
			expectedJSON: `[]`,
		},
		{
			name:        "should fail for invalid output returned by apoc.meta.schema (invalid properties)",
			expectedErr: true,
			mockRecords: []*neo4j.Record{
				{
					Keys: []string{"key", "value"},
					Values: []any{
						"Genre",
						map[string]any{
							"type":          "node",
							"properties":    12,
							"relationships": map[string]any{},
						},
					},
				},
			},
			expectedJSON: `[]`,
		},
		{
			name:        "should fail for invalid output returned by apoc.meta.schema (invalid Node.Relationship.direction)",
			expectedErr: true,
			mockRecords: []*neo4j.Record{
				{
					Keys: []string{"key", "value"},
					Values: []any{
						"Movie",
						map[string]any{
							"type": "node",
							"properties": map[string]any{
								"title":    map[string]any{"type": "STRING", "indexed": false},
								"released": map[string]any{"type": "INTEGER", "indexed": false},
							},
							"relationships": map[string]any{
								"ACTED_IN": map[string]any{
									"direction": 12, "labels": []any{"Person"}, "properties": map[string]any{"roles": map[string]any{"type": "LIST"}},
								},
							},
						},
					},
				},
			},
			expectedJSON: `[]`,
		},
		{
			name:        "should fail for invalid output returned by apoc.meta.schema (invalid Node.relationship)",
			expectedErr: true,
			mockRecords: []*neo4j.Record{
				{
					Keys: []string{"key", "value"},
					Values: []any{
						"Movie",
						map[string]any{
							"type": "node",
							"properties": map[string]any{
								"title":    map[string]any{"type": "STRING", "indexed": false},
								"released": map[string]any{"type": "INTEGER", "indexed": false},
							},
							"relationships": map[string]any{
								"ACTED_IN": "something",
							},
						},
					},
				},
			},
			expectedJSON: `[]`,
		},
		{
			name:        "should fail for invalid output returned by apoc.meta.schema (invalid Node.relationship labels)",
			expectedErr: true,
			mockRecords: []*neo4j.Record{
				{
					Keys: []string{"key", "value"},
					Values: []any{
						"Movie",
						map[string]any{
							"type": "node",
							"properties": map[string]any{
								"title":    map[string]any{"type": "STRING", "indexed": false},
								"released": map[string]any{"type": "INTEGER", "indexed": false},
							},
							"relationships": map[string]any{
								"ACTED_IN": map[string]any{
									"direction": "in", "labels": "not-valid", "properties": map[string]any{
										"role": map[string]any{"type": "STRING", "indexed": false},
									},
								},
							},
						},
					},
				},
			},
			expectedJSON: `[]`,
		},
		{
			name:        "should fail for invalid output returned by apoc.meta.schema (invalid relationship properties)",
			expectedErr: true,
			mockRecords: []*neo4j.Record{
				{
					Keys: []string{"key", "value"},
					Values: []any{
						"ACTED_IN",
						map[string]any{
							"type":       "relationship",
							"labels":     []any{"Person"},
							"properties": "not-valid",
						},
					},
				},
			},
			expectedJSON: `[]`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockDB := db.NewMockService(ctrl)
			mockDB.EXPECT().
				ExecuteReadQuery(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(tc.mockRecords, nil)

			deps := &tools.ToolDependencies{
				DBService:        mockDB,
				AnalyticsService: analyticsService,
			}

			handler := cypher.GetSchemaHandler(deps, 100)
			result, err := handler(context.Background(), mcp.CallToolRequest{})

			if err != nil {
				t.Fatalf("Expected no error, got: %v", err)
			}

			if result == nil || result.IsError {
				if tc.expectedErr {
					return
				}
				t.Fatal("Expected success result")
			}

			textContent, ok := result.Content[0].(mcp.TextContent)
			if !ok {
				t.Fatal("Expected result content to be TextContent")
			}

			var expectedData, actualData any
			if err := json.Unmarshal([]byte(tc.expectedJSON), &expectedData); err != nil {
				t.Fatalf("failed to unmarshal expected JSON: %v", err)
			}
			if err := json.Unmarshal([]byte(textContent.Text), &actualData); err != nil {
				t.Fatalf("failed to unmarshal actual JSON: %v", err)
			}

			expectedFormatted, _ := json.MarshalIndent(expectedData, "", "  ")
			actualFormatted, _ := json.MarshalIndent(actualData, "", "  ")

			if string(expectedFormatted) != string(actualFormatted) {
				t.Errorf("Expected JSON:\n%s\nGot JSON:\n%s", string(expectedFormatted), string(actualFormatted))
			}
		})
	}
}
*/
