package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"
)

const (
	dynamicPortStart = 20_000
	dynamicPortCount = 10_000
	pollInterval     = 5 * time.Second
	portalsDirectory = ".amp/portals"
)

var (
	threadHostPattern = regexp.MustCompile(`^t-([0-9a-f-]{36})-p[0-9]+\.(.+)$`)
	portalNamePattern = regexp.MustCompile(`[^a-zA-Z0-9-]+`)
)

type deploymentRoute struct {
	deploymentID    string
	sourceHostname  string
	projectSlug     string
	appSlug         string
	environmentSlug string
	workspaceSlug   string
}

type activePortal struct {
	DeploymentID   string `json:"deployment_id"`
	SourceHostname string `json:"source_hostname"`
	Name           string `json:"name"`
	Port           int    `json:"port"`
	URL            string `json:"url"`
	Title          string `json:"title"`
}

type portalProcess struct {
	command *exec.Cmd
	done    chan struct{}
}

type managedPortal struct {
	portal  activePortal
	process *portalProcess
}

type portalManager struct {
	threadID     string
	portalDomain string

	mu        sync.Mutex
	portals   map[string]managedPortal
	lastError string
}

type portalManifest struct {
	Links []struct {
		URL string `json:"url"`
	} `json:"links"`
}

func runDeploymentPortals(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: amp-orb-services deployment-portals <listen-port>")
	}
	listenPort, err := parsePort(args[0])
	if err != nil {
		return err
	}
	threadID, portalDomain, err := findPortalContext(nil)
	if err != nil {
		return err
	}

	manager := &portalManager{
		threadID:     threadID,
		portalDomain: portalDomain,
		mu:           sync.Mutex{},
		portals:      make(map[string]managedPortal),
		lastError:    "",
	}
	go manager.watch()

	handler := http.HandlerFunc(manager.handleStatus)
	return http.ListenAndServe(fmt.Sprintf(":%d", listenPort), handler) //nolint:gosec
}

func findPortalContext(publicURLs []string) (string, string, error) {
	deadline := time.Now().Add(time.Minute)
	for {
		urls := append([]string(nil), publicURLs...)
		if len(urls) == 0 {
			manifests, err := filepath.Glob(filepath.Join(portalsDirectory, "*.json"))
			if err != nil {
				return "", "", err
			}
			sort.Strings(manifests)
			for _, path := range manifests {
				content, readErr := os.ReadFile(path)
				if readErr != nil {
					continue
				}
				var manifest portalManifest
				if json.Unmarshal(content, &manifest) != nil {
					continue
				}
				for _, link := range manifest.Links {
					urls = append(urls, link.URL)
				}
			}
		}

		for _, publicURL := range urls {
			parsed, err := url.Parse(publicURL)
			if err != nil {
				continue
			}
			match := threadHostPattern.FindStringSubmatch(parsed.Hostname())
			if len(match) == 3 {
				return "T-" + match[1], match[2], nil
			}
		}

		if len(publicURLs) > 0 || time.Now().After(deadline) {
			return "", "", fmt.Errorf("no Amp thread portal hostname is available")
		}
		time.Sleep(time.Second)
	}
}

func (m *portalManager) watch() {
	cleaned := false
	for {
		var err error
		if !cleaned {
			err = cleanStaleManifests()
			cleaned = err == nil
		}
		if err == nil {
			var routes []deploymentRoute
			routes, err = listDeploymentRoutes()
			if err == nil {
				err = m.reconcile(routes)
			}
		}
		m.setError(err)
		time.Sleep(pollInterval)
	}
}

func listDeploymentRoutes() ([]deploymentRoute, error) {
	pod, err := runKubectl(
		"-n", "unkey",
		"get", "pod",
		"-l", "app=mysql",
		"-o", "jsonpath={.items[0].metadata.name}",
	)
	if err != nil {
		return nil, err
	}
	pod = strings.TrimSpace(pod)
	if pod == "" {
		return nil, nil
	}

	query := `
SELECT
  route.deployment_id,
  route.fully_qualified_domain_name,
  project.slug,
  app.slug,
  environment.slug,
  workspace.slug
FROM frontline_routes AS route
JOIN deployments AS deployment ON deployment.id = route.deployment_id
JOIN environments AS environment ON environment.id = route.environment_id
JOIN apps AS app ON app.id = route.app_id
JOIN projects AS project ON project.id = route.project_id
JOIN workspaces AS workspace ON workspace.id = project.workspace_id
WHERE route.sticky = 'deployment'
  AND route.fully_qualified_domain_name LIKE '%.unkey.local'
  AND deployment.status = 'ready'
  AND deployment.desired_state = 'running'
  AND EXISTS (
    SELECT 1
    FROM instances AS instance
    WHERE instance.deployment_id = deployment.id
      AND instance.status = 'running'
  )
ORDER BY route.deployment_id, route.created_at;
`
	result, err := runKubectl(
		"-n", "unkey", "exec", pod,
		"--", "env", "MYSQL_PWD=password",
		"mysql", "-N", "-u", "unkey", "unkey", "-e", query,
	)
	if err != nil {
		return nil, err
	}

	routes := make([]deploymentRoute, 0)
	seen := make(map[string]struct{})
	for _, line := range strings.Split(strings.TrimSpace(result), "\n") {
		fields := strings.Split(line, "\t")
		if len(fields) != 6 {
			continue
		}
		if _, ok := seen[fields[0]]; ok || !hostnamePattern.MatchString(fields[1]) {
			continue
		}
		routes = append(routes, deploymentRoute{
			deploymentID:    fields[0],
			sourceHostname:  fields[1],
			projectSlug:     fields[2],
			appSlug:         fields[3],
			environmentSlug: fields[4],
			workspaceSlug:   fields[5],
		})
		seen[fields[0]] = struct{}{}
	}
	return routes, nil
}

func (m *portalManager) reconcile(routes []deploymentRoute) error {
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
		stopManagedPortal(managed)
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
		port, err := allocatePort(route.deploymentID, usedPorts)
		if err != nil {
			return err
		}
		managed, err := m.startPortal(route, port)
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

func (m *portalManager) startPortal(route deploymentRoute, port int) (managedPortal, error) {
	name := portalName(route.deploymentID)
	title := portalTitle(route)
	publicURL := fmt.Sprintf(
		"https://t-%s-p%d.%s/",
		strings.ToLower(strings.TrimPrefix(m.threadID, "T-")),
		port,
		m.portalDomain,
	)

	executable, err := os.Executable()
	if err != nil {
		return managedPortal{}, err
	}
	command := exec.Command(executable, "frontline", fmt.Sprint(port), publicURL, route.sourceHostname)
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

	manifestPath := filepath.Join(portalsDirectory, name+".json")
	cleanup := func() {
		stopPortalProcess(process)
		_ = os.Remove(manifestPath)
	}
	if err = waitUntilListening(port, process); err != nil {
		cleanup()
		return managedPortal{}, err
	}

	output, err := runAmpPortal(
		m.threadID,
		m.portalDomain,
		port,
		name,
		title,
		fmt.Sprintf("%s deployment %s routed through Frontline.", route.workspaceSlug, route.deploymentID),
	)
	if err != nil {
		cleanup()
		return managedPortal{}, err
	}
	portalURL := ""
	for _, line := range strings.Split(output, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "https://") {
			portalURL = strings.TrimSpace(line)
		}
	}
	if portalURL == "" {
		cleanup()
		return managedPortal{}, fmt.Errorf("Amp did not return a portal URL for %s", name)
	}

	portal := activePortal{
		DeploymentID:   route.deploymentID,
		SourceHostname: route.sourceHostname,
		Name:           name,
		Port:           port,
		URL:            portalURL,
		Title:          title,
	}
	fmt.Printf("Deployment portal ready: %s -> %s\n", title, portalURL)
	return managedPortal{portal: portal, process: process}, nil
}

func runAmpPortal(threadID, domain string, port int, name, title, description string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
	defer cancel()

	command := exec.CommandContext(
		ctx,
		homePath(".amp", "bin", "amp"),
		"orb", "portal", fmt.Sprint(port),
		"--thread", threadID,
		"--domain", domain,
		"--name", name,
		"--title", title,
		"--description", description,
	)
	command.Env = portalEnvironment(threadID, domain)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	err := command.Run()
	if err != nil {
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = strings.TrimSpace(stdout.String())
		}
		return "", fmt.Errorf("amp orb portal failed: %s", message)
	}
	return stdout.String(), nil
}

func portalEnvironment(threadID, domain string) []string {
	environment := make([]string, 0, len(os.Environ())+2)
	for _, value := range os.Environ() {
		if strings.HasPrefix(value, "AMP_THREAD_ID=") || strings.HasPrefix(value, "AMP_PORTAL_DOMAIN=") {
			continue
		}
		environment = append(environment, value)
	}
	return append(environment, "AMP_THREAD_ID="+threadID, "AMP_PORTAL_DOMAIN="+domain)
}

func allocatePort(deploymentID string, used map[int]struct{}) (int, error) {
	digest := sha256.Sum256([]byte(deploymentID))
	offset := int(binary.BigEndian.Uint32(digest[:4])) % dynamicPortCount
	for step := range dynamicPortCount {
		port := dynamicPortStart + (offset+step)%dynamicPortCount
		if _, ok := used[port]; ok {
			continue
		}
		listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
		if err != nil {
			continue
		}
		_ = listener.Close()
		return port, nil
	}
	return 0, fmt.Errorf("no deployment portal ports are available")
}

func waitUntilListening(port int, process *portalProcess) error {
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if !process.alive() {
			return fmt.Errorf("deployment portal process exited")
		}
		connection, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 200*time.Millisecond)
		if err == nil {
			_ = connection.Close()
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("deployment portal did not listen on port %d", port)
}

func (p *portalProcess) alive() bool {
	select {
	case <-p.done:
		return false
	default:
		return true
	}
}

func stopPortalProcess(process *portalProcess) {
	if !process.alive() {
		return
	}
	_ = process.command.Process.Signal(syscall.SIGTERM)
	select {
	case <-process.done:
		return
	case <-time.After(5 * time.Second):
		_ = process.command.Process.Kill()
		<-process.done
	}
}

func stopManagedPortal(managed managedPortal) {
	stopPortalProcess(managed.process)
	_ = os.Remove(filepath.Join(portalsDirectory, managed.portal.Name+".json"))
}

func portalName(deploymentID string) string {
	suffix := portalNamePattern.ReplaceAllString(deploymentID, "-")
	suffix = strings.ToLower(strings.Trim(suffix, "-"))
	return "deployment-id-" + suffix
}

func portalTitle(route deploymentRoute) string {
	app := ""
	if route.appSlug != "default" {
		app = "/" + route.appSlug
	}
	return fmt.Sprintf("%s%s · %s · %s", route.projectSlug, app, route.environmentSlug, route.deploymentID)
}

func cleanStaleManifests() error {
	if err := os.MkdirAll(portalsDirectory, 0o755); err != nil {
		return err
	}
	for _, pattern := range []string{"deployment-env-*.json", "deployment-id-*.json"} {
		manifests, err := filepath.Glob(filepath.Join(portalsDirectory, pattern))
		if err != nil {
			return err
		}
		for _, manifest := range manifests {
			if err = os.Remove(manifest); err != nil && !os.IsNotExist(err) {
				return err
			}
		}
	}
	for _, name := range []string{"deployment.json", "deployment-portals.json"} {
		if err := os.Remove(filepath.Join(portalsDirectory, name)); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

func (m *portalManager) setError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	message := ""
	if err != nil {
		message = err.Error()
	}
	if message != "" && message != m.lastError {
		fmt.Printf("Unable to reconcile deployment portals: %s\n", message)
	}
	m.lastError = message
}

func (m *portalManager) handleStatus(response http.ResponseWriter, _ *http.Request) {
	m.mu.Lock()
	portals := make([]activePortal, 0, len(m.portals))
	for _, managed := range m.portals {
		portals = append(portals, managed.portal)
	}
	errorMessage := m.lastError
	m.mu.Unlock()
	sort.Slice(portals, func(i, j int) bool {
		return portals[i].DeploymentID < portals[j].DeploymentID
	})

	response.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(response).Encode(map[string]any{
		"status":  "ok",
		"portals": portals,
		"error":   errorMessage,
	}); err != nil {
		return
	}
}
