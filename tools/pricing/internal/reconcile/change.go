package reconcile

import (
	"fmt"
	"sort"
	"strings"
)

// Action is what the reconciler will do (plan mode) or did (apply mode) to one
// object.
type Action string

const (
	// ActionNoop: Stripe already matches the catalog.
	ActionNoop Action = "noop"
	// ActionCreate: the object does not exist and will be / was created.
	ActionCreate Action = "create"
	// ActionUpdate: an in-place field (e.g. default_price, enabled_events) differs.
	ActionUpdate Action = "update"
	// ActionReprice: a new immutable price is created, the lookup_key transferred
	// onto it, and the old price archived (never deleted).
	ActionReprice Action = "reprice"
	// ActionOrphan: a managed-namespace object exists in Stripe but is absent from
	// the catalog. Reported, never touched. `verify` fails on these.
	ActionOrphan Action = "orphan"
)

// Change is a single reconciler decision.
type Change struct {
	Action Action
	Kind   string // plan | meter | usage_price | api_product | webhook
	Key    string // lookup_key or identifying key
	Detail string // human-readable, e.g. "$25 -> $30"
}

// IsChange reports whether this entry represents real work (anything but noop).
func (c Change) IsChange() bool { return c.Action != ActionNoop }

// writesStripe reports whether applying this change performs a Stripe write.
// Noops already match the catalog; orphans are report-only.
func (c Change) writesStripe() bool {
	switch c.Action {
	case ActionCreate, ActionUpdate, ActionReprice:
		return true
	default:
		return false
	}
}

// Line renders one change as a single diff row (marker, kind, key, detail),
// tinted by action when color is true. Shared by Render and apply streaming so
// the lines apply prints match the plan exactly.
func (c Change) Line(color bool) string {
	line := fmt.Sprintf("  %s %-12s %-24s %s", marker(c.Action), c.Kind, c.Key, c.Detail)
	line = strings.TrimRight(line, " ")
	if code := colorFor(c.Action, color); code != "" {
		line = code + line + ansiReset
	}
	return line
}

// Result is the full set of decisions from a reconcile pass.
type Result struct {
	Changes []Change
}

func (r *Result) add(c Change) { r.Changes = append(r.Changes, c) }

// HasChanges reports whether anything other than noop is present.
func (r *Result) HasChanges() bool {
	for _, c := range r.Changes {
		if c.IsChange() {
			return true
		}
	}
	return false
}

// HasWrites reports whether applying would perform any Stripe write. Unlike
// HasChanges it excludes orphans, which apply leaves untouched.
func (r *Result) HasWrites() bool {
	for _, c := range r.Changes {
		if c.writesStripe() {
			return true
		}
	}
	return false
}

// Orphans returns the managed-namespace objects missing from the catalog.
func (r *Result) Orphans() []Change {
	var out []Change
	for _, c := range r.Changes {
		if c.Action == ActionOrphan {
			out = append(out, c)
		}
	}
	return out
}

// CountChanges returns the number of non-noop entries.
func (r *Result) CountChanges() int {
	n := 0
	for _, c := range r.Changes {
		if c.IsChange() {
			n++
		}
	}
	return n
}

// marker is the single-character glyph shown in front of a change line:
// + add, ~ change, ! attention.
func marker(a Action) string {
	switch a {
	case ActionCreate:
		return "+"
	case ActionReprice, ActionUpdate:
		return "~"
	case ActionOrphan:
		return "!"
	default:
		return " "
	}
}

// Render produces the operator-facing diff: a one-line summary, then one
// stable-sorted line per changed object. Unchanged objects are counted, not
// listed. When color is true, each line is tinted by action (green create,
// yellow change, red orphan).
func (r *Result) Render(color bool) string {
	var creates, changes, orphans, unchanged int
	rows := make([]Change, 0, len(r.Changes))
	for _, c := range r.Changes {
		switch c.Action {
		case ActionCreate:
			creates++
		case ActionUpdate, ActionReprice:
			changes++
		case ActionOrphan:
			orphans++
		default:
			unchanged++
			continue // noop: counted, not listed
		}
		rows = append(rows, c)
	}
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].Kind != rows[j].Kind {
			return rows[i].Kind < rows[j].Kind
		}
		return rows[i].Key < rows[j].Key
	})

	var b strings.Builder
	b.WriteString(summaryLine(creates, changes, orphans, unchanged))
	b.WriteByte('\n')
	if len(rows) == 0 {
		return b.String()
	}
	b.WriteByte('\n')
	for _, c := range rows {
		b.WriteString(c.Line(color))
		b.WriteByte('\n')
	}
	return b.String()
}

// ApplySummary is the one-line, past-tense report shown after an apply, e.g.
// "Apply complete: 2 created, 1 changed.". Orphans are report-only and were left
// untouched.
func (r *Result) ApplySummary() string {
	var created, changed, orphans int
	for _, c := range r.Changes {
		switch c.Action {
		case ActionCreate:
			created++
		case ActionUpdate, ActionReprice:
			changed++
		case ActionOrphan:
			orphans++
		}
	}
	parts := []string{
		fmt.Sprintf("%d created", created),
		fmt.Sprintf("%d changed", changed),
	}
	if orphans > 0 {
		parts = append(parts, fmt.Sprintf("%s untouched", plural(orphans, "orphan", "orphans")))
	}
	return "Apply complete: " + strings.Join(parts, ", ") + "."
}

func summaryLine(creates, changes, orphans, unchanged int) string {
	if creates == 0 && changes == 0 && orphans == 0 {
		return fmt.Sprintf("No changes. %s match the catalog.", plural(unchanged, "object", "objects"))
	}
	parts := []string{
		fmt.Sprintf("%d to create", creates),
		fmt.Sprintf("%d to change", changes),
	}
	if orphans > 0 {
		parts = append(parts, plural(orphans, "orphan", "orphans"))
	}
	parts = append(parts, fmt.Sprintf("%d unchanged", unchanged))
	return "Plan: " + strings.Join(parts, ", ") + "."
}

func plural(n int, one, many string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, one)
	}
	return fmt.Sprintf("%d %s", n, many)
}
