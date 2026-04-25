package edgeredirect

import (
	"net/http"
	"testing"

	edgeredirectv1 "github.com/unkeyed/unkey/gen/proto/frontline/edgeredirect/v1"
)

func ruleHTTPS(id string, status int32, enabled bool) *edgeredirectv1.Rule {
	return &edgeredirectv1.Rule{
		Id:      id,
		Enabled: enabled,
		Status:  status,
		Kind:    &edgeredirectv1.Rule_RequireHttps{RequireHttps: &edgeredirectv1.RequireHTTPS{}},
	}
}

func ruleStripWWW(id string, enabled bool) *edgeredirectv1.Rule {
	return &edgeredirectv1.Rule{
		Id:      id,
		Enabled: enabled,
		Kind:    &edgeredirectv1.Rule_StripWww{StripWww: &edgeredirectv1.StripWWW{}},
	}
}

func TestEngineEvaluate_FirstMatchWins(t *testing.T) {
	t.Parallel()
	e := New()
	rules := []*edgeredirectv1.Rule{
		ruleHTTPS("first", 301, true),
		ruleHTTPS("second", 308, true),
	}
	res := e.Evaluate(mkReq(t, "http", "example.com", "/foo"), rules)
	if res == nil {
		t.Fatal("expected match, got nil")
	}
	if res.RuleID != "first" {
		t.Fatalf("rule id = %q, want first", res.RuleID)
	}
	if res.Status != 301 {
		t.Fatalf("status = %d, want 301", res.Status)
	}
}

func TestEngineEvaluate_DisabledRulesSkipped(t *testing.T) {
	t.Parallel()
	e := New()
	rules := []*edgeredirectv1.Rule{
		ruleHTTPS("disabled", 0, false),
		ruleHTTPS("enabled", 0, true),
	}
	res := e.Evaluate(mkReq(t, "http", "example.com", "/"), rules)
	if res == nil {
		t.Fatal("expected match, got nil")
	}
	if res.RuleID != "enabled" {
		t.Fatalf("rule id = %q, want enabled", res.RuleID)
	}
}

func TestEngineEvaluate_NoMatchReturnsNil(t *testing.T) {
	t.Parallel()
	e := New()
	rules := []*edgeredirectv1.Rule{
		ruleStripWWW("strip", true), // request has no www, won't match
	}
	res := e.Evaluate(mkReq(t, "https", "example.com", "/"), rules)
	if res != nil {
		t.Fatalf("expected nil, got %+v", res)
	}
}

func TestEngineEvaluate_EmptyRules(t *testing.T) {
	t.Parallel()
	e := New()
	if res := e.Evaluate(mkReq(t, "http", "example.com", "/"), nil); res != nil {
		t.Fatalf("nil rules -> %+v, want nil", res)
	}
	if res := e.Evaluate(mkReq(t, "http", "example.com", "/"), []*edgeredirectv1.Rule{}); res != nil {
		t.Fatalf("empty rules -> %+v, want nil", res)
	}
}

func TestEngineEvaluate_NilRequestSafe(t *testing.T) {
	t.Parallel()
	e := New()
	if res := e.Evaluate(nil, []*edgeredirectv1.Rule{ruleHTTPS("x", 0, true)}); res != nil {
		t.Fatalf("nil req -> %+v, want nil", res)
	}
}

func TestEngineEvaluate_UnknownKindSkipped(t *testing.T) {
	t.Parallel()
	e := New()
	// A rule with no Kind oneof set should skip silently rather than panic.
	rules := []*edgeredirectv1.Rule{
		{Id: "empty-kind", Enabled: true},
		ruleHTTPS("fallback", 0, true),
	}
	res := e.Evaluate(mkReq(t, "http", "example.com", "/"), rules)
	if res == nil {
		t.Fatal("expected fallback match")
	}
	if res.RuleID != "fallback" {
		t.Fatalf("rule id = %q, want fallback", res.RuleID)
	}
}

func TestEngineEvaluate_NilRuleSkipped(t *testing.T) {
	t.Parallel()
	e := New()
	rules := []*edgeredirectv1.Rule{
		nil,
		ruleHTTPS("fallback", 0, true),
	}
	res := e.Evaluate(mkReq(t, "http", "example.com", "/"), rules)
	if res == nil || res.RuleID != "fallback" {
		t.Fatalf("expected fallback, got %+v", res)
	}
}

func TestEngineEvaluate_ResultRuleKindLabel(t *testing.T) {
	t.Parallel()
	e := New()
	res := e.Evaluate(mkReq(t, "http", "example.com", "/"), []*edgeredirectv1.Rule{ruleHTTPS("x", 0, true)})
	if res == nil {
		t.Fatal("expected match")
	}
	if res.RuleKind != "require_https" {
		t.Fatalf("rule kind = %q, want require_https", res.RuleKind)
	}
}

func TestEngineEvaluate_DefaultStatusIs308(t *testing.T) {
	t.Parallel()
	e := New()
	res := e.Evaluate(mkReq(t, "http", "example.com", "/"), []*edgeredirectv1.Rule{ruleHTTPS("x", 0, true)})
	if res.Status != http.StatusPermanentRedirect {
		t.Fatalf("status = %d, want %d", res.Status, http.StatusPermanentRedirect)
	}
}

func TestEngineEvaluate_AllocsRequireHTTPS(t *testing.T) {
	// Not parallel: AllocsPerRun forbids it.
	e := New()
	rules := []*edgeredirectv1.Rule{ruleHTTPS("x", 0, true)}
	req := mkReq(t, "http", "example.com", "/foo?x=1")
	// Allocation budget: Result struct + builder buffer + location string +
	// the URL.RequestURI() call (which itself escapes the path and concatenates
	// the query). Five is the steady-state baseline; the test guards against
	// regressions, not perfection.
	avg := testing.AllocsPerRun(1000, func() {
		_ = e.Evaluate(req, rules)
	})
	if avg > 5 {
		t.Fatalf("allocs/op = %.2f, want <= 5", avg)
	}
}
