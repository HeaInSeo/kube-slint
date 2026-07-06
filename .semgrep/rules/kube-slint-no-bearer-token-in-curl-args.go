package fixtures

import "fmt"

func bad(token string) string {
	// ruleid: kube-slint-no-bearer-token-in-curl-args
	return fmt.Sprintf("curl -sS -H Authorization: Bearer %s %s", token, "https://example/metrics")
}

func good(insecureFlag, metricsURL string) string {
	// ok: kube-slint-no-bearer-token-in-curl-args
	return fmt.Sprintf(`set -euo pipefail;
TOKEN="$(cat /var/run/secrets/kubernetes.io/serviceaccount/token)";
curl %s -sS --fail-with-body -H "Authorization: Bearer ${TOKEN}" "%s";`, insecureFlag, metricsURL)
}
