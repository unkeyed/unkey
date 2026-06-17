package reconcile

// Minimal ANSI coloring, no dependency. The CLI decides whether to enable it (a
// TTY with NO_COLOR unset); rendering never inspects the environment itself.
const (
	ansiReset  = "\033[0m"
	ansiGreen  = "\033[32m"
	ansiYellow = "\033[33m"
	ansiRed    = "\033[31m"
	ansiDim    = "\033[2m"
)

// colorFor returns the ANSI code for an action, or "" when color is disabled.
func colorFor(a Action, color bool) string {
	if !color {
		return ""
	}

	switch a {
	case ActionCreate:
		return ansiGreen
	case ActionUpdate, ActionReprice:
		return ansiYellow
	case ActionOrphan:
		return ansiRed
	default:
		return ansiDim
	}
}
