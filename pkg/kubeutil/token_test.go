package kubeutil

import (
	"context"
	"os/exec"
	"strings"
	"testing"

	"github.com/HeaInSeo/kube-slint/pkg/slo"
)

type stubRunner struct {
	out string
	err error
}

func (r stubRunner) Run(_ context.Context, _ slo.Logger, _ *exec.Cmd) (string, error) {
	return r.out, r.err
}

func TestRequestServiceAccountTokenOnce_RedactsBodyOnParseFailure(t *testing.T) {
	// Regression test for N3: a malformed/truncated TokenRequest response can
	// still contain a real "token":"..." fragment. The parse-failure error
	// must not leak it verbatim.
	const leakedToken = "eyJhbGciOiJSUzI1NiJ9.super-secret-payload"
	truncated := `{"status":{"token":"` + leakedToken // deliberately unterminated JSON

	r := stubRunner{out: truncated}
	_, err := requestServiceAccountTokenOnce(context.Background(), r, nil, "ns", "sa")
	if err == nil {
		t.Fatal("expected a JSON parse error")
	}
	if strings.Contains(err.Error(), leakedToken) {
		t.Fatalf("token leaked into error message: %v", err)
	}
	if !strings.Contains(err.Error(), "[REDACTED]") {
		t.Fatalf("expected redaction marker in error message: %v", err)
	}
}

func TestRequestServiceAccountTokenOnce_Success(t *testing.T) {
	r := stubRunner{out: `{"status":{"token":"abc123"}}`}
	tok, err := requestServiceAccountTokenOnce(context.Background(), r, nil, "ns", "sa")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tok != "abc123" {
		t.Fatalf("expected token %q, got %q", "abc123", tok)
	}
}
