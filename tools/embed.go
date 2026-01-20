package tools

import (
	"embed"
)

// ConfigFiles embeds all YAML configuration files from the config subdirectory
//
//go:embed all:config
var ConfigFiles embed.FS
