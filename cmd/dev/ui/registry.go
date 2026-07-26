package ui

// tabID identifies a registered pane. New tabs get the next constant and a registry entry.
type tabID int

const (
	tabStack tabID = iota
	tabSeed
	tabStripe
	tabGitHub
	tabProcs
)

// Tab is metadata for the tab bar. Order here is the workflow order shown in the UI.
type Tab struct {
	ID    tabID
	Label string
}

// workflowTabs follows the local dev loop: stack up, seed data, stripe billing, github webhooks.
var workflowTabs = []Tab{
	{ID: tabStack, Label: "Stack"},
	{ID: tabSeed, Label: "Seed"},
	{ID: tabStripe, Label: "Stripe"},
	{ID: tabGitHub, Label: "GitHub"},
	{ID: tabProcs, Label: "Procs"},
}

func newPaneMap() map[tabID]Pane {
	return map[tabID]Pane{
		tabStack:  newStackPane(),
		tabSeed:   newSeedPane(),
		tabStripe: newStripePane(),
		tabGitHub: newGithubPane(),
		tabProcs:  newProcsPane(),
	}
}

func tabIndex(tabs []Tab, id tabID) int {
	for i, t := range tabs {
		if t.ID == id {
			return i
		}
	}
	return 0
}

func tabByHotkey(tabs []Tab, key string) (tabID, bool) {
	for i, t := range tabs {
		if key == hotkeyForIndex(i) {
			return t.ID, true
		}
	}
	return tabStack, false
}

func hotkeyForIndex(i int) string {
	return string(rune('1' + i))
}

func prevTabID(tabs []Tab, active tabID) tabID {
	i := tabIndex(tabs, active)
	if i == 0 {
		return tabs[len(tabs)-1].ID
	}
	return tabs[i-1].ID
}

func nextTabID(tabs []Tab, active tabID) tabID {
	i := tabIndex(tabs, active)
	if i >= len(tabs)-1 {
		return tabs[0].ID
	}
	return tabs[i+1].ID
}
