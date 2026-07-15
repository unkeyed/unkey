package reconcile

import (
	"strings"
	"testing"
)

func sample() *Result {
	return &Result{Changes: []Change{
		{ActionNoop, "plan", "plan.starter", "$5"},
		{ActionCreate, "plan", "plan.business", "$50"},
		{ActionReprice, "plan", "plan.pro", "$25 -> $30"},
		{ActionOrphan, "price", "plan.legacy", "in Stripe, not in catalog"},
	}}
}

func TestRenderPlain(t *testing.T) {
	out := sample().Render(false)

	if strings.Contains(out, "\033[") {
		t.Errorf("plain render must not contain ANSI codes:\n%s", out)
	}
	if !strings.HasPrefix(out, "Plan: 1 to create, 1 to change, 1 orphan, 1 unchanged.") {
		t.Errorf("summary line wrong:\n%s", out)
	}
	// Unchanged objects are counted, not listed.
	if strings.Contains(out, "plan.starter") {
		t.Errorf("noop should not be listed:\n%s", out)
	}
	for _, want := range []string{"+ plan", "~ plan", "! price", "plan.business", "plan.pro", "plan.legacy"} {
		if !strings.Contains(out, want) {
			t.Errorf("render missing %q:\n%s", want, out)
		}
	}
}

func TestRenderColor(t *testing.T) {
	out := sample().Render(true)
	if !strings.Contains(out, ansiGreen+"  + plan") {
		t.Errorf("create line should be green:\n%q", out)
	}
	if !strings.Contains(out, ansiRed+"  ! price") {
		t.Errorf("orphan line should be red:\n%q", out)
	}
	if !strings.Contains(out, ansiReset) {
		t.Errorf("colored lines must reset:\n%q", out)
	}
}

func TestApplySummary(t *testing.T) {
	got := sample().ApplySummary()
	want := "Apply complete: 1 created, 1 changed, 1 orphan untouched."
	if got != want {
		t.Errorf("apply summary:\n got %q\nwant %q", got, want)
	}
}

func TestRenderNoChanges(t *testing.T) {
	r := &Result{Changes: []Change{
		{ActionNoop, "plan", "plan.pro", "$25"},
		{ActionNoop, "meter", "usage.cpu_seconds", "0.0006944¢/unit"},
	}}
	out := r.Render(false)
	if !strings.HasPrefix(out, "No changes. 2 objects match the catalog.") {
		t.Errorf("no-change summary wrong:\n%s", out)
	}
}
