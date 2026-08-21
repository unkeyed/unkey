package main

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"html"
	"net"
	"net/http"
	"os"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	localAppPortStart = 20_000
	localAppPortCount = 10_000
)

// runLocalApps is the Cursor Cloud counterpart to Amp deployment portals.
// It does not call `amp orb portal`. Ready *.unkey.local deployments get a
// localhost port that reaches Frontline with the deployment Host header.
func runLocalApps(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: amp-orb-services local-apps <listen-port>")
	}
	listenPort, err := parsePort(args[0])
	if err != nil {
		return err
	}

	manager := &localAppManager{
		portals: make(map[string]managedPortal),
	}
	go manager.watch()

	handler := http.HandlerFunc(manager.handleStatus)
	return http.ListenAndServe(fmt.Sprintf(":%d", listenPort), handler) //nolint:gosec
}

type localAppManager struct {
	mu        sync.Mutex
	portals   map[string]managedPortal
	lastError string
}

func (m *localAppManager) watch() {
	for {
		routes, err := listDeploymentRoutes()
		if err == nil {
			err = m.reconcile(routes)
		}
		m.setError(err)
		time.Sleep(pollInterval)
	}
}

func (m *localAppManager) reconcile(routes []deploymentRoute) error {
	desired := make(map[string]deploymentRoute, len(routes))
	for _, route := range routes {
		desired[route.deploymentID] = route
	}

	m.mu.Lock()
	current := make(map[string]managedPortal, len(m.portals))
	for deploymentID, portal := range m.portals {
		current[deploymentID] = portal
	}
	m.mu.Unlock()

	for deploymentID, managed := range current {
		route, ok := desired[deploymentID]
		if ok && route.sourceHostname == managed.portal.SourceHostname && managed.process.alive() {
			continue
		}
		stopPortalProcess(managed.process)
		m.mu.Lock()
		delete(m.portals, deploymentID)
		m.mu.Unlock()
	}

	m.mu.Lock()
	usedPorts := make(map[int]struct{}, len(m.portals))
	activeIDs := make(map[string]struct{}, len(m.portals))
	for deploymentID, managed := range m.portals {
		usedPorts[managed.portal.Port] = struct{}{}
		activeIDs[deploymentID] = struct{}{}
	}
	m.mu.Unlock()

	sort.Slice(routes, func(i, j int) bool {
		return routes[i].deploymentID < routes[j].deploymentID
	})
	for _, route := range routes {
		if _, ok := activeIDs[route.deploymentID]; ok {
			continue
		}
		port, err := allocateLocalAppPort(route.deploymentID, usedPorts)
		if err != nil {
			return err
		}
		managed, err := startLocalApp(route, port)
		if err != nil {
			return err
		}
		usedPorts[port] = struct{}{}
		m.mu.Lock()
		m.portals[route.deploymentID] = managed
		m.mu.Unlock()
	}
	return nil
}

func startLocalApp(route deploymentRoute, port int) (managedPortal, error) {
	executable, err := os.Executable()
	if err != nil {
		return managedPortal{}, err
	}
	command := exec.Command(executable, "host-frontline", fmt.Sprint(port), route.sourceHostname)
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	if err = command.Start(); err != nil {
		return managedPortal{}, err
	}
	process := &portalProcess{command: command, done: make(chan struct{})}
	go func() {
		_ = command.Wait()
		close(process.done)
	}()

	if err = waitUntilListening(port, process); err != nil {
		stopPortalProcess(process)
		return managedPortal{}, err
	}

	title := portalTitle(route)
	localURL := fmt.Sprintf("http://127.0.0.1:%d/", port)
	fmt.Printf("Deployment app ready: %s -> %s (%s)\n", title, localURL, route.sourceHostname)
	return managedPortal{
		portal: activePortal{
			DeploymentID:   route.deploymentID,
			SourceHostname: route.sourceHostname,
			Name:           portalName(route.deploymentID),
			Port:           port,
			URL:            localURL,
			Title:          title,
		},
		process: process,
	}, nil
}

func allocateLocalAppPort(deploymentID string, used map[int]struct{}) (int, error) {
	digest := sha256.Sum256([]byte(deploymentID))
	offset := int(binary.BigEndian.Uint32(digest[:4])) % localAppPortCount
	for step := range localAppPortCount {
		port := localAppPortStart + (offset+step)%localAppPortCount
		if _, ok := used[port]; ok {
			continue
		}
		listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err != nil {
			continue
		}
		_ = listener.Close()
		return port, nil
	}
	return 0, fmt.Errorf("no local app ports are available in %d-%d", localAppPortStart, localAppPortStart+localAppPortCount-1)
}

func (m *localAppManager) setError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	message := ""
	if err != nil {
		message = err.Error()
	}
	if message != "" && message != m.lastError {
		fmt.Printf("Unable to reconcile local apps: %s\n", message)
	}
	m.lastError = message
}

func (m *localAppManager) snapshot() ([]activePortal, string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	portals := make([]activePortal, 0, len(m.portals))
	for _, managed := range m.portals {
		portals = append(portals, managed.portal)
	}
	sort.Slice(portals, func(i, j int) bool {
		return portals[i].DeploymentID < portals[j].DeploymentID
	})
	return portals, m.lastError
}

func (m *localAppManager) handleStatus(response http.ResponseWriter, request *http.Request) {
	portals, errorMessage := m.snapshot()
	listed := make([]activePortal, len(portals))
	for i, portal := range portals {
		portal.URL = appPublicURL(request, portal.Port)
		listed[i] = portal
	}
	if request.URL.Path == "/json" || request.Header.Get("Accept") == "application/json" {
		response.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(response).Encode(map[string]any{
			"status": "ok",
			"apps":   listed,
			"error":  errorMessage,
		})
		return
	}

	response.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = fmt.Fprint(response, `<!doctype html><html lang="en"><head><meta charset="utf-8"><title>Deployed apps</title>`)
	_, _ = fmt.Fprint(response, `<meta http-equiv="refresh" content="5">`)
	_, _ = fmt.Fprint(response, `<style>body{font:16px/1.4 sans-serif;margin:2rem;max-width:52rem}a{color:#0b57d0}li{margin:.4rem 0}.err{color:#a40000}code{font-size:.95em}</style></head><body>`)
	_, _ = fmt.Fprint(response, `<h1>Deployed apps</h1>`)
	_, _ = fmt.Fprint(response, `<p>Each ready <code>*.unkey.local</code> deployment is bridged through Frontline on a localhost port, the Cursor counterpart to Amp deployment portals.</p>`)
	if errorMessage != "" {
		_, _ = fmt.Fprintf(response, `<p class="err">%s</p>`, html.EscapeString(errorMessage))
	}
	if len(listed) == 0 {
		_, _ = fmt.Fprint(response, `<p>No ready deployments yet.</p>`)
	} else {
		_, _ = fmt.Fprint(response, `<ul>`)
		for _, portal := range listed {
			_, _ = fmt.Fprintf(
				response,
				`<li><a href="%s">%s</a> on <code>localhost:%d</code> → <code>%s</code></li>`,
				html.EscapeString(portal.URL),
				html.EscapeString(portal.Title),
				portal.Port,
				html.EscapeString(portal.SourceHostname),
			)
		}
		_, _ = fmt.Fprint(response, `</ul>`)
	}
	_, _ = fmt.Fprint(response, `</body></html>`)
}

func appPublicURL(request *http.Request, port int) string {
	scheme := "http"
	if request.TLS != nil || strings.EqualFold(request.Header.Get("X-Forwarded-Proto"), "https") {
		scheme = "https"
	}
	host := request.Host
	if host == "" {
		return fmt.Sprintf("http://127.0.0.1:%d/", port)
	}
	hostname, _, err := net.SplitHostPort(host)
	if err != nil {
		return fmt.Sprintf("http://127.0.0.1:%d/", port)
	}
	return fmt.Sprintf("%s://%s/", scheme, net.JoinHostPort(hostname, fmt.Sprint(port)))
}
