package fixtures

type client struct {
	TLSInsecureSkipVerify    bool
	DangerouslySkipTLSVerify bool
}

func badLiteral() *client {
	// ruleid: kube-slint-no-insecure-skip-verify
	return &client{TLSInsecureSkipVerify: true}
}

func badAssign(c *client) {
	// ruleid: kube-slint-no-insecure-skip-verify
	c.TLSInsecureSkipVerify = true
}

func goodDangerouslyNamed() *client {
	// ok: kube-slint-no-insecure-skip-verify
	return &client{DangerouslySkipTLSVerify: true}
}

func goodDefaultFalse() *client {
	// ok: kube-slint-no-insecure-skip-verify
	return &client{TLSInsecureSkipVerify: false}
}
