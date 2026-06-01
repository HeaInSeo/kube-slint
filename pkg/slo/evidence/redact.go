package evidence

import "regexp"

// redactPatterns matches secret-bearing substrings.
// Each pattern must have exactly one capturing group that is the non-secret prefix;
// the replacement appends [REDACTED] after that group.
var redactPatterns = []*regexp.Regexp{
	// "Bearer <token>" anywhere — covers both full header strings and map values
	regexp.MustCompile(`(?i)(Bearer\s+)\S+`),
	// key=value: token, password, passwd, secret
	regexp.MustCompile(`(?i)(\b(?:token|password|passwd|secret)\s*=\s*)\S+`),
}

// RedactString replaces known secret patterns in s with [REDACTED].
// Safe to call on empty strings or strings with no secrets.
func RedactString(s string) string {
	for _, re := range redactPatterns {
		s = re.ReplaceAllString(s, "${1}[REDACTED]")
	}
	return s
}

// RedactMap returns a copy of m with all values passed through RedactString.
// Keys are not modified.
func RedactMap(m map[string]string) map[string]string {
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = RedactString(v)
	}
	return out
}
