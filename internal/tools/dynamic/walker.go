package dynamic

import (
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// EmbeddedFS is a package-level variable that can be set with embedded config files
var EmbeddedFS embed.FS

// WalkConfigDirectory walks the config directory and loads all YAML tool definitions
// It first attempts to load from embedded filesystem, falling back to OS filesystem if needed
func WalkConfigDirectory(configDir string) ([]*ToolConfig, error) {
	// Try embedded filesystem first if it's set
	configs, err := walkEmbeddedConfigs()
	if err == nil && len(configs) > 0 {
		slog.Info("loaded tools from embedded filesystem", "count", len(configs))
		return configs, nil
	}

	// Fall back to OS filesystem (for development/testing)
	return walkOSFilesystem(configDir)
}

// walkEmbeddedConfigs loads tools from the embedded filesystem
func walkEmbeddedConfigs() ([]*ToolConfig, error) {
	var configs []*ToolConfig

	// Check if embedded FS is available
	// Try to stat a known path to see if FS has content
	if _, err := fs.Stat(EmbeddedFS, "."); err != nil {
		// Embedded FS not available or empty
		return nil, fmt.Errorf("embedded FS not available")
	}

	// Walk through all embedded files
	err := fs.WalkDir(EmbeddedFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if d.IsDir() {
			return nil
		}

		// Only process YAML files
		if !strings.HasSuffix(d.Name(), ".yaml") && !strings.HasSuffix(d.Name(), ".yml") {
			return nil
		}

		// Read embedded file
		data, err := EmbeddedFS.ReadFile(path)
		if err != nil {
			slog.Error("failed to read embedded config", "path", path, "error", err)
			return err
		}

		// Parse and validate
		config, err := parseToolConfig(data, path)
		if err != nil {
			slog.Error("failed to parse embedded tool config", "path", path, "error", err)
			return err
		}

		configs = append(configs, config)
		slog.Info("loaded tool config from embedded FS", "tool", config.Name, "category", config.Metadata.Category, "path", path)

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk embedded configs: %w", err)
	}

	return configs, nil
}

// walkOSFilesystem walks the OS filesystem (fallback for development)
func walkOSFilesystem(configDir string) ([]*ToolConfig, error) {
	var configs []*ToolConfig

	// Check if config directory exists
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		slog.Warn("config directory does not exist", "dir", configDir)
		return configs, nil // Return empty slice, not an error
	}

	err := filepath.Walk(configDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			slog.Error("error accessing path", "path", path, "error", err)
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Only process YAML files
		if !strings.HasSuffix(info.Name(), ".yaml") && !strings.HasSuffix(info.Name(), ".yml") {
			return nil
		}

		// Read file from OS
		data, err := os.ReadFile(path)
		if err != nil {
			slog.Error("failed to read config file", "path", path, "error", err)
			return err
		}

		// Get relative path for category derivation
		relPath, _ := filepath.Rel(configDir, path)

		// Parse and validate
		config, err := parseToolConfig(data, relPath)
		if err != nil {
			slog.Error("failed to parse tool config", "path", path, "error", err)
			return err
		}

		configs = append(configs, config)
		slog.Info("loaded tool config from filesystem", "tool", config.Name, "category", config.Metadata.Category, "path", path)

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk config directory: %w", err)
	}

	return configs, nil
}

// parseToolConfig parses and validates a YAML tool configuration
func parseToolConfig(data []byte, path string) (*ToolConfig, error) {
	// Parse YAML
	var config ToolConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Derive category from directory structure
	category := deriveCategoryFromPath(path)
	config.Metadata.Category = category

	// Validate required fields
	if config.Name == "" {
		return nil, fmt.Errorf("tool name is required in config file: %s", path)
	}

	if config.Title == "" {
		return nil, fmt.Errorf("tool title is required in config file: %s", path)
	}

	if config.Description == "" {
		return nil, fmt.Errorf("tool description is required in config file: %s", path)
	}

	// If execution block exists, validate it
	if config.Execution != nil {
		if config.Execution.Mode == "" {
			return nil, fmt.Errorf("execution mode is required when execution block is present in config file: %s", path)
		}

		if config.Execution.Mode != "read" && config.Execution.Mode != "write" {
			return nil, fmt.Errorf("execution mode must be 'read' or 'write' in config file: %s", path)
		}
	}

	return &config, nil
}

// deriveCategoryFromPath extracts the category from the file path
// Example: "tools/config/fraud/detect-synthetic-identity.yaml" -> "fraud"
func deriveCategoryFromPath(path string) string {
	// Normalize path separators
	path = filepath.ToSlash(path)

	// Split path into components
	parts := strings.Split(path, "/")

	// Find "config" in the path and take the next component
	for i, part := range parts {
		if part == "config" && i+1 < len(parts) {
			return parts[i+1]
		}
	}

	// If we have at least 2 parts, use the first as category
	if len(parts) >= 2 {
		// Skip tools/ if present
		if parts[0] == "tools" && len(parts) >= 3 {
			return parts[1]
		}
		return parts[0]
	}

	return "general"
}
