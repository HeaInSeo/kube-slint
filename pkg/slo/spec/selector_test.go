package spec

import (
	"errors"
	"strings"
	"testing"
)

func TestSelector_CanonicalKeyIsMatcherOrderIndependent(t *testing.T) {
	a := SelectMetric("m", AggregateSum, LabelEq("b", "2"), LabelRegexp("a", "x|y"))
	b := SelectMetric("m", AggregateSum, LabelRegexp("a", "x|y"), LabelEq("b", "2"))
	if a.Key != b.Key {
		t.Fatalf("canonical keys differ: %q vs %q", a.Key, b.Key)
	}
	if !strings.HasPrefix(a.Key, selectorKeyPrefix) {
		t.Fatalf("key %q lacks selector prefix", a.Key)
	}
	// Aggregation is part of the identity.
	c := SelectMetric("m", AggregateNone, LabelEq("b", "2"), LabelRegexp("a", "x|y"))
	if c.Key == a.Key {
		t.Fatal("aggregation must change the canonical key")
	}
}

func TestSelector_ResolveEqualityAndSum(t *testing.T) {
	values := map[string]float64{
		`reconcile_total{controller="a",result="success"}`: 3,
		`reconcile_total{controller="a",result="error"}`:   1,
		`reconcile_total{controller="b",result="success"}`: 100,
		`reconcile_total`: 104, // synthesized bare aggregate must never match
	}
	ref := SelectMetric("reconcile_total", AggregateSum, LabelEq("controller", "a"))
	got, matched, err := ref.Selector.Resolve(values)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got != 4 {
		t.Fatalf("sum = %v, want 4", got)
	}
	want := []string{`reconcile_total{controller="a",result="error"}`, `reconcile_total{controller="a",result="success"}`}
	if strings.Join(matched, "|") != strings.Join(want, "|") {
		t.Fatalf("matched = %v, want sorted %v", matched, want)
	}
}

func TestSelector_RegexpIsAnchored(t *testing.T) {
	values := map[string]float64{
		`rest_client_requests_total{code="500",host="h",method="GET"}`:  2,
		`rest_client_requests_total{code="1500",host="h",method="GET"}`: 7,
		`rest_client_requests_total{code="50",host="h",method="GET"}`:   9,
		`rest_client_requests_total{code="200",host="h",method="GET"}`:  50,
	}
	// Unanchored expression still matches only whole values.
	ref := SelectMetric("rest_client_requests_total", AggregateSum, LabelRegexp("code", "5[0-9][0-9]"))
	got, _, err := ref.Selector.Resolve(values)
	if err != nil || got != 2 {
		t.Fatalf("got %v, %v; want 2, nil", got, err)
	}
}

func TestSelector_MatcherRequiresLabelPresence(t *testing.T) {
	values := map[string]float64{
		`workqueue_depth{controller="c",name="c"}`:             5, // no priority label
		`workqueue_depth{controller="c",name="c",priority=""}`: 1,
	}
	ref := SelectMetric("workqueue_depth", AggregateNone,
		LabelEq("name", "c"), LabelEq("controller", "c"), LabelEq("priority", ""))
	got, _, err := ref.Selector.Resolve(values)
	if err != nil || got != 1 {
		t.Fatalf("got %v, %v; want 1, nil", got, err)
	}
}

func TestSelector_ZeroMatchFailsClosed(t *testing.T) {
	ref := SelectMetric("m", AggregateSum, LabelEq("a", "x"))
	_, _, err := ref.Selector.Resolve(map[string]float64{`m{a="y"}`: 1, `m`: 1, `other{a="x"}`: 1})
	if !errors.Is(err, ErrSelectorNoMatch) {
		t.Fatalf("err = %v, want ErrSelectorNoMatch", err)
	}
}

func TestSelector_UndeclaredMultiMatchFailsClosed(t *testing.T) {
	ref := SelectMetric("m", AggregateNone, LabelEq("a", "x"))
	_, matched, err := ref.Selector.Resolve(map[string]float64{`m{a="x",b="1"}`: 1, `m{a="x",b="2"}`: 2})
	if !errors.Is(err, ErrSelectorAmbiguous) {
		t.Fatalf("err = %v, want ErrSelectorAmbiguous", err)
	}
	if len(matched) != 2 {
		t.Fatalf("matched = %v, want both series reported", matched)
	}
}

func TestSelector_ValidateRejectsMalformed(t *testing.T) {
	cases := map[string]MetricRef{
		"empty metric":      SelectMetric("", AggregateSum, LabelEq("a", "x")),
		"no matchers":       SelectMetric("m", AggregateSum),
		"bad aggregation":   SelectMetric("m", Aggregation("avg"), LabelEq("a", "x")),
		"bad regexp":        SelectMetric("m", AggregateSum, LabelRegexp("a", "(")),
		"empty regexp":      SelectMetric("m", AggregateSum, LabelRegexp("a", "")),
		"bad operator":      SelectMetric("m", AggregateSum, LabelMatcher{Name: "a", Op: "!=", Value: "x"}),
		"empty label name":  SelectMetric("m", AggregateSum, LabelEq("", "x")),
		"duplicate label":   SelectMetric("m", AggregateSum, LabelEq("a", "x"), LabelEq("a", "y")),
		"metric with brace": SelectMetric(`m{a="x"}`, AggregateSum, LabelEq("a", "x")),
	}
	for name, ref := range cases {
		t.Run(name, func(t *testing.T) {
			if err := ref.Validate(); err == nil {
				t.Fatal("Validate must reject malformed selector")
			}
			if _, _, err := ref.Selector.Resolve(map[string]float64{`m{a="x"}`: 1}); err == nil {
				t.Fatal("Resolve must reject malformed selector")
			}
		})
	}
}

func TestMetricRef_ValidateRejectsKeyDrift(t *testing.T) {
	ref := SelectMetric("m", AggregateSum, LabelEq("a", "x"))
	ref.Key = `m{a="x"}`
	if err := ref.Validate(); err == nil {
		t.Fatal("a selector ref whose Key is not canonical must be rejected")
	}
}

func TestMetricRef_ExactKeyUnchanged(t *testing.T) {
	ref := PromMetric("m", Labels{"a": "x"})
	if ref.Selector != nil {
		t.Fatal("PromMetric must stay an exact-key ref")
	}
	if ref.IdentityKey() != ref.Key || ref.Key != `m{a="x"}` {
		t.Fatalf("exact-key identity changed: %q", ref.IdentityKey())
	}
	if err := ref.Validate(); err != nil {
		t.Fatalf("exact-key ref must stay valid: %v", err)
	}
}

func TestSLISpec_ValidateSelectorsRejectsWindowModes(t *testing.T) {
	s := SLISpec{
		ID:      "w",
		Inputs:  []MetricRef{SelectMetric("m", AggregateSum, LabelEq("a", "x"))},
		Compute: ComputeSpec{Mode: ComputeWindowAvg},
	}
	if err := s.ValidateSelectors(); err == nil {
		t.Fatal("selector inputs must be rejected for window modes")
	}
	s.Compute.Mode = ComputeDelta
	if err := s.ValidateSelectors(); err != nil {
		t.Fatalf("delta mode must accept a valid selector: %v", err)
	}
}

func TestSelector_EmptyMatchZeroIfFamilyPresent(t *testing.T) {
	ref := SelectMetric("rest_client_requests_total", AggregateSum, LabelRegexp("code", "5[0-9][0-9]")).
		WithEmptyMatch(EmptyMatchZeroIfFamilyPresent)
	healthy := map[string]float64{`rest_client_requests_total{code="200",host="h",method="GET"}`: 10}
	got, matched, err := ref.Selector.Resolve(healthy)
	if err != nil || got != 0 || len(matched) != 0 {
		t.Fatalf("family present, no match: got %v %v %v; want 0, none, nil", got, matched, err)
	}
	// Only the synthesized bare aggregate is present: the family is present.
	if got, _, err := ref.Selector.Resolve(map[string]float64{`rest_client_requests_total`: 10}); err != nil || got != 0 {
		t.Fatalf("bare aggregate only: got %v, %v; want 0, nil", got, err)
	}
	// Family absent stays missing.
	_, _, err = ref.Selector.Resolve(map[string]float64{`other_total{code="500"}`: 1})
	if !errors.Is(err, ErrSelectorFamilyAbsent) {
		t.Fatalf("err = %v, want ErrSelectorFamilyAbsent", err)
	}
	// Matches still sum as before.
	healthy[`rest_client_requests_total{code="500",host="h",method="GET"}`] = 2
	if got, _, err := ref.Selector.Resolve(healthy); err != nil || got != 2 {
		t.Fatalf("got %v, %v; want 2, nil", got, err)
	}
}

func TestSelector_EmptyMatchDefaultSkips(t *testing.T) {
	ref := SelectMetric("m", AggregateSum, LabelEq("a", "x"))
	if ref.Selector.EmptyMatch != EmptyMatchSkip {
		t.Fatalf("default policy = %q, want skip", ref.Selector.EmptyMatch)
	}
	_, _, err := ref.Selector.Resolve(map[string]float64{`m{a="y"}`: 1})
	if !errors.Is(err, ErrSelectorNoMatch) {
		t.Fatalf("err = %v, want ErrSelectorNoMatch", err)
	}
}

func TestSelector_EmptyMatchPolicyValidation(t *testing.T) {
	cases := map[string]MetricRef{
		"zero without sum": SelectMetric("m", AggregateNone, LabelEq("a", "x")).WithEmptyMatch(EmptyMatchZeroIfFamilyPresent),
		"unknown policy":   SelectMetric("m", AggregateSum, LabelEq("a", "x")).WithEmptyMatch(EmptyMatchPolicy("zero")),
		"metric w/ space":  SelectMetric("empty=zero_if_family_present m", AggregateSum, LabelEq("a", "x")),
		"label w/ dash":    SelectMetric("m", AggregateSum, LabelEq("a-b", "x")),
	}
	for name, ref := range cases {
		t.Run(name, func(t *testing.T) {
			if err := ref.Validate(); err == nil {
				t.Fatal("Validate must reject")
			}
		})
	}

	s := SLISpec{ID: "z", Compute: ComputeSpec{Mode: ComputeEnd},
		Inputs: []MetricRef{SelectMetric("m", AggregateSum, LabelEq("a", "x")).WithEmptyMatch(EmptyMatchZeroIfFamilyPresent)}}
	if err := s.ValidateSelectors(); err == nil {
		t.Fatal("zero_if_family_present must be rejected outside delta mode")
	}
	s.Compute.Mode = ComputeDelta
	if err := s.ValidateSelectors(); err != nil {
		t.Fatalf("delta+sum zero policy must validate: %v", err)
	}
}

func TestSelector_EmptyMatchPolicyIsInIdentity(t *testing.T) {
	skip := SelectMetric("m", AggregateSum, LabelEq("a", "x"))
	zero := skip.WithEmptyMatch(EmptyMatchZeroIfFamilyPresent)
	if skip.Key == zero.Key || skip.IdentityKey() == zero.IdentityKey() {
		t.Fatalf("empty-match policy must change the identity: %q", zero.Key)
	}
	if skip.Selector.EmptyMatch != EmptyMatchSkip {
		t.Fatal("WithEmptyMatch must not mutate the receiver")
	}
	if err := zero.Validate(); err != nil {
		t.Fatalf("WithEmptyMatch must keep Key canonical: %v", err)
	}
}

func TestMetricRef_ExactAndSelectorIdentitiesDisjoint(t *testing.T) {
	sel := SelectMetric("m", AggregateSum, LabelEq("a", "x"))
	zero := sel.WithEmptyMatch(EmptyMatchZeroIfFamilyPresent)
	for _, s := range []MetricRef{sel, zero} {
		exact := InputKey(s.Key)
		if exact.IdentityKey() == s.IdentityKey() {
			t.Fatalf("exact key %q shares the selector identity", s.Key)
		}
		if strings.HasPrefix(exact.IdentityKey(), selectorKeyPrefix) {
			t.Fatalf("exact identity %q is in the selector namespace", exact.IdentityKey())
		}
	}
	// Escaping stays injective across exact keys.
	a := InputKey(sel.Key)
	b := InputKey(exactIdentityPrefix + sel.Key)
	if a.IdentityKey() == b.IdentityKey() {
		t.Fatal("distinct exact keys must keep distinct identities")
	}
}

// Codex P2 4126142138: the sum must not depend on how a fetcher serializes
// labels. With 1e16, -1e16 and 1 the float result depends on addition order:
// canonical order a,b,c gives 1, while raw-key order a,c,b would give 0.
func TestSelector_SumOrderIsCanonicalNotRaw(t *testing.T) {
	a, b, c := `m{app="x",s="a"}`, `m{app="x",s="b"}`, `m{app="x",s="c"}`
	ref := SelectMetric("m", AggregateSum, LabelEq("app", "x"))
	for name, keys := range map[string][3]string{
		"canonical":            {a, b, c},
		"all reversed":         {`m{s="a",app="x"}`, `m{s="b",app="x"}`, `m{s="c",app="x"}`},
		"b reversed":           {a, `m{s="b",app="x"}`, c},
		"a reversed, b spaced": {`m{s="a",app="x"}`, `m{app="x", s="b"}`, c},
	} {
		t.Run(name, func(t *testing.T) {
			values := map[string]float64{keys[0]: 1e16, keys[1]: -1e16, keys[2]: 1}
			got, matched, err := ref.Selector.Resolve(values)
			if err != nil {
				t.Fatalf("Resolve: %v", err)
			}
			if got != 1 {
				t.Fatalf("sum = %v, want 1 (canonical a,b,c order)", got)
			}
			// Raw keys are returned, ordered by canonical identity.
			if strings.Join(matched, "|") != strings.Join(keys[:], "|") {
				t.Fatalf("matched = %v, want raw keys in canonical order %v", matched, keys)
			}
		})
	}
}
