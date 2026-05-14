package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.PreferLinuxEngine != "spd-say" {
		t.Errorf("expected default PreferLinuxEngine=spd-say, got %q", cfg.PreferLinuxEngine)
	}
	if cfg.TestText == "" {
		t.Error("expected non-empty default TestText")
	}
}

func TestLoadConfig_NonExistent(t *testing.T) {
	cfg, err := LoadConfig("/does/not/exist.json")
	if err != nil {
		t.Fatalf("expected no error for missing config, got: %v", err)
	}
	// Should fall back to defaults.
	def := DefaultConfig()
	if cfg.PreferLinuxEngine != def.PreferLinuxEngine {
		t.Errorf("got %q; want %q", cfg.PreferLinuxEngine, def.PreferLinuxEngine)
	}
}

func TestLoadConfig_ValidFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	content := `{"voice":"Alice","rate":"500","prefer_linux_engine":"espeak","test_text":"Test"}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Voice != "Alice" {
		t.Errorf("voice: got %q; want Alice", cfg.Voice)
	}
	if cfg.Rate != "500" {
		t.Errorf("rate: got %q; want 500", cfg.Rate)
	}
}

func TestLoadConfig_InvalidRate(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte(`{"rate":"fast"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadConfig(path)
	if err == nil {
		t.Error("expected validation error for non-numeric rate, got nil")
	}
}

func TestCreateDefaultConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	if err := CreateDefaultConfig(path); err != nil {
		t.Fatalf("CreateDefaultConfig: %v", err)
	}

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("loading created config: %v", err)
	}
	def := DefaultConfig()
	if cfg.TestText != def.TestText {
		t.Errorf("TestText: got %q; want %q", cfg.TestText, def.TestText)
	}
}
