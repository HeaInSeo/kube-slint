package harness

import (
"os"
"path/filepath"
"testing"
)

func TestDiscoverConfig_Disabled(t *testing.T) {
os.Setenv("SLINT_DISABLE_DISCOVERY", "1")
defer os.Unsetenv("SLINT_DISABLE_DISCOVERY")

_, source, err := DiscoverConfig("")
if err != nil {
t.Fatalf("expected no error, got %v", err)
}
if !source.Disabled {
t.Errorf("expected source to be marked disabled")
}
if source.Type != "injected" {
t.Errorf("expected source type 'injected', got %v", source.Type)
}
}

func TestDiscoverConfig_EnvPath(t *testing.T) {
tmpDir := t.TempDir()
configPath := filepath.Join(tmpDir, "custom.yaml")
os.WriteFile(configPath, []byte("format: v4.4\n"), 0644)

os.Setenv("SLINT_CONFIG_PATH", configPath)
defer os.Unsetenv("SLINT_CONFIG_PATH")

cfg, source, err := DiscoverConfig("")
if err != nil {
t.Fatalf("expected no error, got %v", err)
}
if source.Type != "env" {
t.Errorf("expected source type 'env', got %v", source.Type)
}
if source.Path != configPath {
t.Errorf("expected path %s, got %s", configPath, source.Path)
}
if cfg.Format != "v4.4" {
t.Errorf("expected format 'v4.4', got %s", cfg.Format)
}
}

func TestDiscoverConfig_AutoDiscovery(t *testing.T) {
tmpDir := t.TempDir()
configPath := filepath.Join(tmpDir, ".slint.yaml")
os.WriteFile(configPath, []byte(`
format: v4.4
strictness:
  mode: StrictCollection
`), 0644)

cfg, source, err := DiscoverConfig(tmpDir)
if err != nil {
t.Fatalf("expected no error, got %v", err)
}
if source.Type != "discovered" {
t.Errorf("expected source type 'discovered', got %v", source.Type)
}
if cfg.Format != "v4.4" {
t.Errorf("expected format 'v4.4', got %s", cfg.Format)
}
if cfg.Strictness.Mode != "StrictCollection" {
t.Errorf("expected Strictness.Mode 'StrictCollection', got %s", cfg.Strictness.Mode)
}
}
