// Package index serves the route listing at GET /. Unlike the other
// probes, it needs to know what routes exist — main.go calls Register
// once per route during startup, then mounts Handler at the root.
package index

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
)

type route struct{ Method, Path, Description string }

var routes []route

// Register adds a route to the listing. Call it from main.go for each
// probe so the index at GET / can show it. Pattern is a ServeMux
// pattern like "GET /hello".
func Register(pattern, description string) {
	m, p, _ := strings.Cut(pattern, " ")
	routes = append(routes, route{m, p, description})
}

// Handler renders the registered routes as plain text. Registered by
// main.go.
func Handler(w http.ResponseWriter, r *http.Request) {
	sorted := make([]route, len(routes))
	copy(sorted, routes)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Path != sorted[j].Path {
			return sorted[i].Path < sorted[j].Path
		}
		return sorted[i].Method < sorted[j].Method
	})

	var b strings.Builder
	b.WriteString("\n")
	b.WriteString("  kitchensink — stdlib HTTP server for testing platform features\n")
	b.WriteString("  " + strings.Repeat("─", 62) + "\n\n")
	for _, rt := range sorted {
		fmt.Fprintf(&b, "    %-5s  %s\n           %s\n\n", rt.Method, rt.Path, rt.Description)
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte(b.String()))
}
