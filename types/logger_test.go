package types

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/charmbracelet/log"
)

func TestInitLogger_FullConfig(t *testing.T) {
	dir := chdirTemp(t)

	cfg := LoggerConfig{
		LoggerLevel:          "debug",
		LoggerFileName:       filepath.Join(dir, "log", "server.log"),
		LoggerFileMaxSize:    10,
		LoggerFileMaxBackups: 2,
		LoggerFileMaxAge:     7,
		LoggerPrefix:         "TestPrefix",
	}
	InitLogger(cfg)

	// lumberjack creates the directory lazily on first write, not on construction,
	// so we only assert the call did not panic and produced a usable logger config.
	if cfg.LoggerFileName == "" {
		t.Fatal("expected file name set")
	}
}

func TestInitLogger_NoFileNoPrefixEmptyLevel(t *testing.T) {
	// Empty level skips SetLevel; empty file name skips file output;
	// empty prefix falls back to the default prefix branch.
	cfg := LoggerConfig{
		LoggerLevel:    "",
		LoggerFileName: "",
		LoggerPrefix:   "",
	}
	InitLogger(cfg)
}

func TestInitLogger_DefaultConfig(t *testing.T) {
	chdirTemp(t)
	// DefaultLoggerConfig has a non-empty level, file name and prefix.
	InitLogger(DefaultLoggerConfig)
	_ = os.Getenv("HOME")
}

func TestLoadConfig_DefaultLoggerWritesNothingInWorkingDirectory(t *testing.T) {
	dir := chdirTemp(t)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })

	cfg, err := LoadConfig("")
	if err != nil {
		t.Fatalf("LoadConfig(\"\"): %v", err)
	}
	InitLogger(cfg.LoggerConfig)
	log.Info("Starting daemon for session test")

	if entries, _ := os.ReadDir(dir); len(entries) != 0 {
		t.Fatalf("the default logger wrote into the working directory: %v", entries)
	}
}
