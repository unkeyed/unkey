// Package tui renders styled, aligned terminal output for CLI commands.
//
// It covers the non-interactive half of terminal UX: colors, aligned tables,
// and key-value blocks. For interactive input (menus, pickers) see pkg/prompt.
// Like pkg/prompt it sticks to the standard library plus golang.org/x/term,
// avoiding heavier TUI dependencies.
//
// Styling degrades gracefully: escape codes are emitted only when the writer
// is a terminal, NO_COLOR is unset, and TERM is not "dumb". The same code
// path written to a pipe or file produces plain aligned text, so command
// output stays grep-friendly.
//
// # Usage
//
//	out := tui.New(os.Stdout)
//	out.Println(out.Bold("local") + "  " + out.Dim("clock_123"))
//
//	out.KV().Indent(2).
//		Add("status", out.Green("ready")).
//		Add("frozen at", "2026-06-10T19:52:51Z").
//		Print()
//
//	out.Table("CUSTOMER", "WORKSPACE").
//		Row("cus_123", "ws_local").
//		Print()
package tui
