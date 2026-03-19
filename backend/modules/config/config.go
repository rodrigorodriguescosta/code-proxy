package config

import (
	"os"
	"path/filepath"
)

type Config struct {
	Port         string
	DataDir      string
	DBPath       string
	DefaultModel string
	WorkDir      string // Working directory for CLI providers
	UseACP       bool   // Use ACP instead of exec("claude")
	ACPCommand   string // ACP subprocess command
	ACPArgs      string // Extra arguments for ACP subprocess
}

func Load() *Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "3456"
	}

	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		home, _ := os.UserHomeDir()
		dataDir = filepath.Join(home, ".code-proxy")
	}
	os.MkdirAll(dataDir, 0755)

	defaultModel := os.Getenv("CLAUDE_MODEL")
	if defaultModel == "" {
		defaultModel = "sonnet"
	}

	workDir := os.Getenv("WORK_DIR")
	if workDir == "" {
		workDir, _ = os.Getwd()
	}

	useACP := os.Getenv("USE_ACP") == "true"
	acpCommand := os.Getenv("ACP_COMMAND")
	acpArgs := os.Getenv("ACP_ARGS")

	return &Config{
		Port:         port,
		DataDir:      dataDir,
		DBPath:       filepath.Join(dataDir, "data.db"),
		DefaultModel: defaultModel,
		WorkDir:      workDir,
		UseACP:       useACP,
		ACPCommand:   acpCommand,
		ACPArgs:      acpArgs,
	}
}
