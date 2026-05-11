package slint

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadServiceAccountToken_OK(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "token")
	if err := os.WriteFile(p, []byte("  my-bearer-token\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	tok, err := ReadServiceAccountToken(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tok != "my-bearer-token" {
		t.Fatalf("got %q, want %q", tok, "my-bearer-token")
	}
}

func TestReadServiceAccountToken_Missing(t *testing.T) {
	_, err := ReadServiceAccountToken("/no/such/file/token")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestReadServiceAccountToken_Empty(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "token")
	if err := os.WriteFile(p, []byte("   "), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := ReadServiceAccountToken(p)
	if err == nil {
		t.Fatal("expected error for empty token file")
	}
}

func TestReadServiceAccountTokenFromEnv_EnvSet(t *testing.T) {
	t.Setenv("SLINT_SA_TOKEN_TEST", "env-token")
	tok, err := ReadServiceAccountTokenFromEnv("SLINT_SA_TOKEN_TEST", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tok != "env-token" {
		t.Fatalf("got %q, want %q", tok, "env-token")
	}
}

func TestReadServiceAccountTokenFromEnv_FallbackFile(t *testing.T) {
	t.Setenv("SLINT_SA_TOKEN_UNSET", "")
	dir := t.TempDir()
	p := filepath.Join(dir, "token")
	if err := os.WriteFile(p, []byte("file-token"), 0o600); err != nil {
		t.Fatal(err)
	}
	tok, err := ReadServiceAccountTokenFromEnv("SLINT_SA_TOKEN_UNSET", p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tok != "file-token" {
		t.Fatalf("got %q, want %q", tok, "file-token")
	}
}
