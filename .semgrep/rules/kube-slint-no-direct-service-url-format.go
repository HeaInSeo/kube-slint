package fixtures

import "fmt"

type client struct {
	ServiceURLFormat string
}

func bad(c *client, svc, ns string) string {
	// ruleid: kube-slint-no-direct-service-url-format
	return fmt.Sprintf(c.ServiceURLFormat, svc, ns)
}

func good(c *client, svc, ns string, allowExternal bool) (string, error) {
	// ok: kube-slint-no-direct-service-url-format
	return ValidateMetricsURL(c.ServiceURLFormat, svc, ns, allowExternal)
}

func ValidateMetricsURL(format, service, namespace string, allowExternal bool) (string, error) {
	return "", nil
}
