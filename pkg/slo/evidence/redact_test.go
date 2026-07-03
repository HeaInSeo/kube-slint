package evidence_test

import (
	"testing"

	"github.com/HeaInSeo/kube-slint/pkg/slo/evidence"
	"github.com/stretchr/testify/assert"
)

func TestRedactString(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "bearer token",
			input: "Authorization: Bearer eyJhbGciOiJSUzI1NiJ9.abc",
			want:  "Authorization: Bearer [REDACTED]",
		},
		{
			name:  "bearer token case insensitive",
			input: "authorization: bearer supersecrettoken",
			want:  "authorization: bearer [REDACTED]",
		},
		{
			name:  "token= form",
			input: "token=abc123secret",
			want:  "token=[REDACTED]",
		},
		{
			name:  "password= form",
			input: "password=hunter2",
			want:  "password=[REDACTED]",
		},
		{
			name:  "secret= form",
			input: "secret=my-secret-value",
			want:  "secret=[REDACTED]",
		},
		{
			name:  "passwd= form",
			input: "passwd=abc",
			want:  "passwd=[REDACTED]",
		},
		{
			name:  "no secret — unchanged",
			input: "collectionStatus=Complete",
			want:  "collectionStatus=Complete",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "multiple secrets in one string",
			input: "token=abc Authorization: Bearer xyz",
			want:  "token=[REDACTED] Authorization: Bearer [REDACTED]",
		},
		// Regression cases for N3: JSON-quoted, CLI-flag, and YAML/plain-colon
		// forms weren't covered before — only "Bearer ..." and "key=value".
		{
			name:  "JSON-quoted token field",
			input: `{"status":{"token":"eyJhbGciOiJSUzI1NiJ9.abc"}}`,
			want:  `{"status":{"token":"[REDACTED]"}}`,
		},
		{
			name:  "JSON-quoted token field with spacing",
			input: `{"token" : "abc123secret"}`,
			want:  `{"token" : "[REDACTED]"}`,
		},
		{
			name:  "JSON-quoted clientSecret field",
			input: `{"clientSecret":"hunter2"}`,
			want:  `{"clientSecret":"[REDACTED]"}`,
		},
		{
			name:  "--token flag with space",
			input: "kubectl --token abc123secret get pods",
			want:  "kubectl --token [REDACTED] get pods",
		},
		{
			name:  "--client-key-data flag with =",
			input: "--client-key-data=LS0tLS1CRUdJTg==",
			want:  "--client-key-data=[REDACTED]",
		},
		{
			name:  "--certificate-authority-data flag",
			input: "--certificate-authority-data LS0tLS1CRUdJTg==",
			want:  "--certificate-authority-data [REDACTED]",
		},
		{
			name:  "YAML plain-colon token field (embedded kubeconfig)",
			input: "token: eyJhbGciOiJSUzI1NiJ9.abc",
			want:  "token: [REDACTED]",
		},
		{
			name:  "YAML plain-colon client-key-data field",
			input: "client-key-data: LS0tLS1CRUdJTg==",
			want:  "client-key-data: [REDACTED]",
		},
		{
			name:  "serviceAccountToken key=value form",
			input: "serviceAccountToken=eyJhbGciOiJSUzI1NiJ9.abc",
			want:  "serviceAccountToken=[REDACTED]",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := evidence.RedactString(tc.input)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestRedactMap(t *testing.T) {
	input := map[string]string{
		"Authorization": "Bearer supersecret",
		"X-Request-ID":  "req-123",
		"token":         "mysecrettoken",
	}
	got := evidence.RedactMap(input)

	assert.Equal(t, "Bearer [REDACTED]", got["Authorization"])
	assert.Equal(t, "req-123", got["X-Request-ID"])
	assert.Equal(t, "mysecrettoken", got["token"]) // key "token" is not a value pattern
}

func TestRedactMap_NilSafe(t *testing.T) {
	got := evidence.RedactMap(nil)
	assert.NotNil(t, got)
	assert.Empty(t, got)
}
