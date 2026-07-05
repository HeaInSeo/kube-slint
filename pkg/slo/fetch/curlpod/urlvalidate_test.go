package curlpod

import (
	"strings"
	"testing"
)

const defaultFormat = "https://%s.%s.svc:8443/metrics"

func TestValidateMetricsURL_ClusterLocalSVC_Allowed(t *testing.T) {
	got, err := ValidateMetricsURL(defaultFormat, "hello-operator", "hello-system", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "https://hello-operator.hello-system.svc:8443/metrics" {
		t.Fatalf("unexpected URL: %s", got)
	}
}

func TestValidateMetricsURL_ClusterLocalFQDN_Allowed(t *testing.T) {
	format := "https://%s.%s.svc.cluster.local:8443/metrics"
	if _, err := ValidateMetricsURL(format, "svc", "ns", false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateMetricsURL_PlainHTTPDev_Allowed(t *testing.T) {
	format := "http://%s.%s.svc:8080/metrics"
	if _, err := ValidateMetricsURL(format, "svc", "ns", false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// security/external-service-url.yaml
func TestValidateMetricsURL_ExternalHost_Rejected(t *testing.T) {
	format := "https://evil.example.com/collect?svc=%s&ns=%s"
	_, err := ValidateMetricsURL(format, "svc", "ns", false)
	if err == nil {
		t.Fatal("expected external host to be rejected")
	}
}

func TestValidateMetricsURL_ExternalSubdomainOfService_Rejected(t *testing.T) {
	format := "https://%s.%s.evil.example/metrics"
	_, err := ValidateMetricsURL(format, "svc", "ns", false)
	if err == nil {
		t.Fatal("expected external host to be rejected")
	}
}

// security/ftp-service-url.yaml
func TestValidateMetricsURL_UnsupportedScheme_Rejected(t *testing.T) {
	format := "ftp://%s.%s.svc/metrics"
	_, err := ValidateMetricsURL(format, "svc", "ns", false)
	if err == nil {
		t.Fatal("expected ftp scheme to be rejected")
	}
	if !strings.Contains(err.Error(), "scheme") {
		t.Fatalf("expected scheme-related error, got: %v", err)
	}
}

func TestValidateMetricsURL_RawIP_Rejected(t *testing.T) {
	format := "https://10.0.0.10/metrics"
	// %s verbs unused by this format string; Sprintf still succeeds (extra args ignored... actually
	// fmt.Sprintf with fewer verbs than args appends "%!(EXTRA ...)" — this itself makes url.Parse
	// see a malformed/unexpected host, which must still not be accepted as cluster-local.
	_, err := ValidateMetricsURL(format, "svc", "ns", false)
	if err == nil {
		t.Fatal("expected raw external IP to be rejected")
	}
}

// security/external-service-url-template-injection.yaml
func TestValidateMetricsURL_TemplateInjectionViaServiceName_Rejected(t *testing.T) {
	_, err := ValidateMetricsURL(defaultFormat, "x.evil.example", "ns", false)
	if err == nil {
		t.Fatal("expected non-DNS-label service name to be rejected")
	}
}

func TestValidateMetricsURL_TemplateInjectionViaNamespace_Rejected(t *testing.T) {
	_, err := ValidateMetricsURL(defaultFormat, "svc", "evil.example/../x", false)
	if err == nil {
		t.Fatal("expected non-DNS-label namespace to be rejected")
	}
}

func TestValidateMetricsURL_EmptyServiceOrNamespace_Rejected(t *testing.T) {
	if _, err := ValidateMetricsURL(defaultFormat, "", "ns", false); err == nil {
		t.Fatal("expected empty service name to be rejected")
	}
	if _, err := ValidateMetricsURL(defaultFormat, "svc", "", false); err == nil {
		t.Fatal("expected empty namespace to be rejected")
	}
}

func TestValidateMetricsURL_DangerouslyAllowExternalMetricsURL_Opt(t *testing.T) {
	format := "https://evil.example.com/collect?svc=%s&ns=%s"
	got, err := ValidateMetricsURL(format, "svc", "ns", true)
	if err != nil {
		t.Fatalf("expected explicit opt-in to allow external host, got error: %v", err)
	}
	if !strings.Contains(got, "evil.example.com") {
		t.Fatalf("unexpected URL: %s", got)
	}
}

func TestValidateMetricsURL_DangerouslyAllowExternalMetricsURL_StillRejectsBadScheme(t *testing.T) {
	format := "ftp://%s.%s.svc/metrics"
	_, err := ValidateMetricsURL(format, "svc", "ns", true)
	if err == nil {
		t.Fatal("expected unsupported scheme to be rejected even with external URLs allowed")
	}
}

// security/kube-system-target.yaml
func TestIsDangerousNamespace(t *testing.T) {
	cases := map[string]bool{
		"kube-system":     true,
		"kube-public":     true,
		"kube-node-lease": true,
		"hello-system":    false,
		"default":         false,
		"":                false,
	}
	for ns, want := range cases {
		if got := isDangerousNamespace(ns); got != want {
			t.Errorf("isDangerousNamespace(%q) = %v, want %v", ns, got, want)
		}
	}
}
