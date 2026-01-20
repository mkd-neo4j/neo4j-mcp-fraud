package models

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mkd-neo4j/neo4j-mcp-fraud/internal/tools"
)

const (
	httpTimeout = 10 * time.Second
)

var (
	defaultReferenceModelURLs = []string{
		"https://neo4j.com/developer/industry-use-cases/_attachments/transaction-base-model.txt",
		"https://neo4j.com/developer/industry-use-cases/_attachments/fraud-event-sequence-model.txt",
	}
)

// GetReferenceModelsHandler returns a handler function for the get-data-models tool
func GetReferenceModelsHandler(deps *tools.ToolDependencies) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleGetReferenceModels(ctx, deps, request)
	}
}

// handleGetReferenceModels fetches and returns Neo4j reference data models
func handleGetReferenceModels(ctx context.Context, deps *tools.ToolDependencies, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if deps.AnalyticsService == nil {
		errMessage := "analytics service is not initialized"
		slog.Error(errMessage)
		return mcp.NewToolResultError(errMessage), nil
	}

	deps.AnalyticsService.EmitEvent(deps.AnalyticsService.NewToolsEvent("get-data-models"))

	slog.Info("fetching Neo4j reference data models")

	// Fetch reference models from default URLs
	var referenceModels []string
	referenceModelURLs := defaultReferenceModelURLs

	// Fetch models from URLs
	for _, url := range referenceModelURLs {
		content, err := fetchReferenceModelFromURL(ctx, url)
		if err != nil {
			slog.Warn("failed to fetch reference model from URL", "url", url, "error", err)
			continue
		}
		referenceModels = append(referenceModels, fmt.Sprintf("=== Reference Model from %s ===\n%s", url, content))
	}

	// Combine all reference models
	var combinedReferenceModel string
	if len(referenceModels) > 0 {
		combinedReferenceModel = strings.Join(referenceModels, "\n\n")
	} else {
		slog.Warn("no reference models could be loaded")
		return mcp.NewToolResultError("Failed to fetch reference models from Neo4j"), nil
	}

	// Truncate to prevent timeout (max 15KB)
	truncated := truncateReferenceModel(combinedReferenceModel, 15000)

	slog.Info("returning reference models", "size", len(truncated))

	return mcp.NewToolResultText(truncated), nil
}

// fetchReferenceModelFromURL fetches a reference model from a URL
func fetchReferenceModelFromURL(ctx context.Context, url string) (string, error) {
	client := &http.Client{
		Timeout: httpTimeout,
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	return string(body), nil
}

// truncateReferenceModel truncates the reference model to a maximum size to prevent response timeouts
func truncateReferenceModel(referenceModel string, maxChars int) string {
	if len(referenceModel) <= maxChars {
		return referenceModel
	}

	truncated := referenceModel[:maxChars]
	lastNewline := strings.LastIndex(truncated, "\n")
	if lastNewline > maxChars-500 {
		truncated = truncated[:lastNewline]
	}

	return truncated + "\n\n...[Reference models truncated for size - full models available at neo4j.com/developer]..."
}
