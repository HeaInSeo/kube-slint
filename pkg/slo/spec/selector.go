package spec

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/HeaInSeo/kube-slint/pkg/slo/common/promkey"
)

// selectorKeyPrefix marks MetricRef.Key values that encode a Selector rather
// than a raw input key. The canonical selector key is what appears in
// InputsUsed/InputsMissing and what the SLI measurement identity hashes, so a
// selector input never shares an identity with an exact-key input.
const selectorKeyPrefix = "kube-slint.select/v1 "

// MatchOp is a label matcher operator.
type MatchOp string

const (
	// MatchEqual requires the label to be present with exactly this value.
	MatchEqual MatchOp = "="
	// MatchRegexp requires the label to be present and its whole value to match
	// an RE2 expression. The expression is always anchored at both ends.
	MatchRegexp MatchOp = "=~"
)

// LabelMatcher selects series by one label. A matcher never matches a series
// that does not carry the label, so an unlabeled (e.g. promtext.Aggregate
// synthesized) series can never be selected.
type LabelMatcher struct {
	Name  string
	Op    MatchOp
	Value string
}

// LabelEq returns an equality matcher.
func LabelEq(name, value string) LabelMatcher {
	return LabelMatcher{Name: name, Op: MatchEqual, Value: value}
}

// LabelRegexp returns an anchored RE2 matcher.
func LabelRegexp(name, expr string) LabelMatcher {
	return LabelMatcher{Name: name, Op: MatchRegexp, Value: expr}
}

// Aggregation declares how multiple matched series are combined.
type Aggregation string

const (
	// AggregateNone requires exactly one matched series.
	AggregateNone Aggregation = ""
	// AggregateSum sums every matched series.
	AggregateSum Aggregation = "sum"
)

// Selector identifies input series by metric name plus label matchers instead
// of an exact key. It is only valid for point compute modes
// (single/start/end/delta).
type Selector struct {
	Metric    string
	Matchers  []LabelMatcher
	Aggregate Aggregation
}

var (
	// ErrSelectorNoMatch is returned when no series matches a selector.
	ErrSelectorNoMatch = errors.New("selector matched no series")
	// ErrSelectorAmbiguous is returned when several series match a selector
	// that does not declare an aggregation.
	ErrSelectorAmbiguous = errors.New("selector matched multiple series without declared aggregation")
)

// SelectMetric returns a MetricRef whose value is resolved by a Selector. The
// Key is set to the selector's canonical form.
func SelectMetric(name string, agg Aggregation, matchers ...LabelMatcher) MetricRef {
	sel := &Selector{Metric: name, Matchers: append([]LabelMatcher(nil), matchers...), Aggregate: agg}
	return MetricRef{Key: sel.CanonicalKey(), Selector: sel}
}

// CanonicalKey returns a deterministic key for the selector: matchers are
// sorted by name, operator, then value, and label values are escaped as in
// Prometheus text keys.
func (s *Selector) CanonicalKey() string {
	ms := append([]LabelMatcher(nil), s.Matchers...)
	sort.Slice(ms, func(i, j int) bool {
		if ms[i].Name != ms[j].Name {
			return ms[i].Name < ms[j].Name
		}
		if ms[i].Op != ms[j].Op {
			return ms[i].Op < ms[j].Op
		}
		return ms[i].Value < ms[j].Value
	})

	var b strings.Builder
	b.WriteString(selectorKeyPrefix)
	if s.Aggregate == AggregateNone {
		b.WriteString("one")
	} else {
		b.WriteString(string(s.Aggregate))
	}
	b.WriteByte(' ')
	b.WriteString(s.Metric)
	b.WriteByte('{')
	for i, m := range ms {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(m.Name)
		b.WriteString(string(m.Op))
		b.WriteByte('"')
		b.WriteString(promkey.EscapeLabelValue(m.Value))
		b.WriteByte('"')
	}
	b.WriteByte('}')
	return b.String()
}

// Validate reports a malformed selector. It is called before any measurement
// work so a malformed selector fails the run instead of silently skipping.
func (s *Selector) Validate() error {
	if strings.TrimSpace(s.Metric) == "" || strings.ContainsAny(s.Metric, "{}\"") {
		return fmt.Errorf("selector: invalid metric name %q", s.Metric)
	}
	if len(s.Matchers) == 0 {
		return fmt.Errorf("selector %q: at least one label matcher is required", s.Metric)
	}
	switch s.Aggregate {
	case AggregateNone, AggregateSum:
	default:
		return fmt.Errorf("selector %q: unsupported aggregation %q", s.Metric, s.Aggregate)
	}
	seen := map[string]bool{}
	for _, m := range s.Matchers {
		if strings.TrimSpace(m.Name) == "" || strings.ContainsAny(m.Name, "{}=,\" ") {
			return fmt.Errorf("selector %q: invalid label name %q", s.Metric, m.Name)
		}
		if seen[m.Name] {
			return fmt.Errorf("selector %q: duplicate matcher for label %q", s.Metric, m.Name)
		}
		seen[m.Name] = true
		switch m.Op {
		case MatchEqual:
		case MatchRegexp:
			if _, err := compileAnchored(m.Value); err != nil {
				return fmt.Errorf("selector %q: label %q: %w", s.Metric, m.Name, err)
			}
		default:
			return fmt.Errorf("selector %q: label %q: unsupported operator %q", s.Metric, m.Name, m.Op)
		}
	}
	return nil
}

// Resolve selects the matching series from values and returns their combined
// value together with the sorted matched keys. Keys that are not parseable
// Prometheus keys never match. Matched values are summed in sorted key order,
// so the result does not depend on map iteration order.
func (s *Selector) Resolve(values map[string]float64) (float64, []string, error) {
	if err := s.Validate(); err != nil {
		return 0, nil, err
	}
	res := make([]*regexp.Regexp, len(s.Matchers))
	for i, m := range s.Matchers {
		if m.Op == MatchRegexp {
			res[i], _ = compileAnchored(m.Value) // validated above
		}
	}

	var matched []string
	for key := range values {
		name, labels, err := promkey.Parse(key)
		if err != nil || name != s.Metric {
			continue
		}
		if s.matches(labels, res) {
			matched = append(matched, key)
		}
	}
	sort.Strings(matched)

	switch {
	case len(matched) == 0:
		return 0, nil, ErrSelectorNoMatch
	case len(matched) > 1 && s.Aggregate == AggregateNone:
		return 0, matched, fmt.Errorf("%w (%d series)", ErrSelectorAmbiguous, len(matched))
	}
	var total float64
	for _, k := range matched {
		total += values[k]
	}
	return total, matched, nil
}

func (s *Selector) matches(labels map[string]string, res []*regexp.Regexp) bool {
	for i, m := range s.Matchers {
		v, ok := labels[m.Name]
		if !ok {
			return false
		}
		switch m.Op {
		case MatchEqual:
			if v != m.Value {
				return false
			}
		case MatchRegexp:
			if !res[i].MatchString(v) {
				return false
			}
		default:
			return false
		}
	}
	return true
}

func compileAnchored(expr string) (*regexp.Regexp, error) {
	if expr == "" {
		return nil, fmt.Errorf("empty regexp")
	}
	re, err := regexp.Compile("^(?:" + expr + ")$")
	if err != nil {
		return nil, fmt.Errorf("invalid regexp %q: %w", expr, err)
	}
	return re, nil
}

// IdentityKey returns the key that identifies this input in results and in
// the SLI measurement identity.
func (m MetricRef) IdentityKey() string {
	if m.Selector != nil {
		return m.Selector.CanonicalKey()
	}
	return m.Key
}

// Validate reports a malformed input reference. Exact-key refs are always
// valid (legacy behavior). A selector ref must be well formed and its Key must
// equal the selector's canonical key, so the reported and hashed identity
// cannot drift from what is actually resolved.
func (m MetricRef) Validate() error {
	if m.Selector == nil {
		return nil
	}
	if err := m.Selector.Validate(); err != nil {
		return err
	}
	if want := m.Selector.CanonicalKey(); m.Key != want {
		return fmt.Errorf("selector input key %q does not match canonical key %q (use SelectMetric)", m.Key, want)
	}
	return nil
}

// ValidateSelectors checks every selector input of s. Selector inputs are only
// supported by point compute modes.
func (s SLISpec) ValidateSelectors() error {
	for _, in := range s.Inputs {
		if in.Selector == nil {
			continue
		}
		switch s.Compute.Mode {
		case ComputeSingle, ComputeStart, ComputeEnd, ComputeDelta:
		default:
			return fmt.Errorf("sli %q: selector inputs are not supported for compute mode %q", s.ID, s.Compute.Mode)
		}
		if err := in.Validate(); err != nil {
			return fmt.Errorf("sli %q: %w", s.ID, err)
		}
	}
	return nil
}
