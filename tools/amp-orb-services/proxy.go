package main

import (
	"bytes"
	"crypto/sha256"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"
)

const (
	bufferSize            = 64 * 1024
	frontlineUpstreamPort = 9443
)

var (
	healthPath         = []byte("/_unkey/internal/health/ready")
	headerTerminator   = []byte("\r\n\r\n")
	hostnamePattern    = regexp.MustCompile(`^[a-zA-Z0-9.-]+$`)
	e2bHostnamePattern = regexp.MustCompile(`^[0-9]+-[a-z0-9]+\.e2b\.app$`)
)

// runTCPProxy gives Amp a stable listener for services that minikube publishes on
// a different loopback port. This is a raw TCP proxy instead of an HTTP reverse
// proxy so Connect RPC, streaming responses, and WebSocket upgrades stay intact.
func runTCPProxy(args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("usage: amp-orb-services proxy <listen-port> <upstream-port>")
	}
	listenPort, err := parsePort(args[0])
	if err != nil {
		return err
	}
	upstreamPort, err := parsePort(args[1])
	if err != nil {
		return err
	}

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", listenPort))
	if err != nil {
		return err
	}
	defer func() { _ = listener.Close() }()

	for {
		client, acceptErr := listener.Accept()
		if acceptErr != nil {
			fmt.Printf("Unable to accept proxy connection: %s\n", acceptErr)
			time.Sleep(100 * time.Millisecond)
			continue
		}
		go func() {
			upstream, dialErr := net.DialTimeout(
				"tcp",
				fmt.Sprintf("127.0.0.1:%d", upstreamPort),
				5*time.Second,
			)
			if dialErr != nil {
				_ = client.Close()
				return
			}
			bridgeConnections(client, upstream)
		}()
	}
}

func bridgeConnections(first, second net.Conn) {
	defer func() { _ = first.Close() }()
	defer func() { _ = second.Close() }()

	done := make(chan struct{}, 2)
	copyConnection := func(destination, source net.Conn) {
		_, _ = io.Copy(destination, source)
		done <- struct{}{}
	}
	go copyConnection(first, second)
	go copyConnection(second, first)
	<-done
}

// frontlineProxy keeps deployed application requests on the normal Frontline
// path. Amp terminates public TLS before this process, so the proxy restores TLS
// locally and preserves the public Host header for Frontline route lookup.
type frontlineProxy struct {
	publicHostname string
	sourceHostname string
	registered     map[string]struct{}
	routeMu        sync.Mutex
	tlsConfig      *tls.Config
}

func runFrontlineProxy(args []string) error {
	if len(args) != 3 {
		return fmt.Errorf("usage: amp-orb-services frontline <listen-port> <public-url> <source-hostname>")
	}
	listenPort, err := parsePort(args[0])
	if err != nil {
		return err
	}
	publicURL, err := url.Parse(args[1])
	if err != nil || !hostnamePattern.MatchString(publicURL.Hostname()) {
		return fmt.Errorf("public URL must contain a valid hostname")
	}
	sourceHostname := args[2]
	if !hostnamePattern.MatchString(sourceHostname) {
		return fmt.Errorf("source route must be a valid hostname")
	}

	proxy := &frontlineProxy{
		publicHostname: publicURL.Hostname(),
		sourceHostname: sourceHostname,
		registered:     make(map[string]struct{}),
		routeMu:        sync.Mutex{},
		// Amp terminates public TLS. Frontline uses a self-signed development
		// certificate on loopback, so the bridge must not verify that certificate.
		tlsConfig: &tls.Config{ //nolint:exhaustruct,gosec
			MinVersion:         tls.VersionTLS12,
			ServerName:         sourceHostname,
			InsecureSkipVerify: true,
		},
	}
	go proxy.registerPublicRouteWhenAvailable()

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", listenPort))
	if err != nil {
		return err
	}
	defer func() { _ = listener.Close() }()

	for {
		client, acceptErr := listener.Accept()
		if acceptErr != nil {
			fmt.Printf("Unable to accept Frontline connection: %s\n", acceptErr)
			time.Sleep(100 * time.Millisecond)
			continue
		}
		go proxy.handle(client)
	}
}

// registerRoute copies the immutable deployment route to the hostname that Amp
// assigned to the portal. Frontline intentionally rejects unknown hosts, and Amp
// hostnames do not exist when the deployment route is first created.
func (p *frontlineProxy) registerRoute(hostname string) (bool, error) {
	pod, err := runKubectl(
		"-n", "unkey",
		"get", "pod",
		"-l", "app=mysql",
		"-o", "jsonpath={.items[0].metadata.name}",
	)
	if err != nil {
		return false, err
	}
	pod = strings.TrimSpace(pod)
	if pod == "" {
		return false, nil
	}

	routeID := fmt.Sprintf("flr_orb_%x", sha256.Sum256([]byte(hostname)))[:32]
	// Both hostnames are restricted to letters, digits, dots, and hyphens above.
	query := fmt.Sprintf(`
INSERT INTO frontline_routes (
  id,
  project_id,
  app_id,
  deployment_id,
  environment_id,
  fully_qualified_domain_name,
  sticky,
  created_at,
  updated_at
)
SELECT
  '%s',
  project_id,
  app_id,
  deployment_id,
  environment_id,
  '%s',
  sticky,
  CAST(UNIX_TIMESTAMP(CURRENT_TIMESTAMP(3)) * 1000 AS SIGNED),
  NULL
FROM frontline_routes
WHERE fully_qualified_domain_name = '%s'
ON DUPLICATE KEY UPDATE
  project_id = VALUES(project_id),
  app_id = VALUES(app_id),
  deployment_id = VALUES(deployment_id),
  environment_id = VALUES(environment_id),
  sticky = VALUES(sticky),
  updated_at = CAST(UNIX_TIMESTAMP(CURRENT_TIMESTAMP(3)) * 1000 AS SIGNED);

SELECT COUNT(*)
FROM frontline_routes AS portal
JOIN frontline_routes AS source
  ON source.fully_qualified_domain_name = '%s'
  AND portal.environment_id = source.environment_id
  AND portal.deployment_id = source.deployment_id
  AND portal.sticky = source.sticky
WHERE portal.fully_qualified_domain_name = '%s';
`, routeID, hostname, p.sourceHostname, p.sourceHostname, hostname) //nolint:gosec

	result, err := runKubectl(
		"-n", "unkey", "exec", pod,
		"--", "env", "MYSQL_PWD=password",
		"mysql", "-N", "-u", "unkey", "unkey", "-e", query,
	)
	if err != nil {
		return false, err
	}
	return strings.HasSuffix(strings.TrimSpace(result), "1"), nil
}

func (p *frontlineProxy) ensureRoute(hostname string) bool {
	p.routeMu.Lock()
	defer p.routeMu.Unlock()

	if _, ok := p.registered[hostname]; ok {
		return true
	}
	ready, err := p.registerRoute(hostname)
	if err == nil && ready {
		p.registered[hostname] = struct{}{}
		return true
	}
	return false
}

func (p *frontlineProxy) registerPublicRouteWhenAvailable() {
	for {
		if p.ensureRoute(p.publicHostname) {
			fmt.Printf("Frontline route ready: https://%s -> %s\n", p.publicHostname, p.sourceHostname)
			return
		}
		time.Sleep(5 * time.Second)
	}
}

func (p *frontlineProxy) handle(client net.Conn) {
	defer func() { _ = client.Close() }()
	request, err := readInitialRequest(client)
	if err != nil || len(request) == 0 {
		return
	}

	path := requestPath(request)
	hostname := requestHostname(request)
	if !bytes.Equal(path, healthPath) && (!p.isPortalHostname(hostname) || !p.ensureRoute(hostname)) {
		body := []byte(fmt.Sprintf("Waiting for deployment route %s\n", p.sourceHostname))
		_, _ = fmt.Fprintf(
			client,
			"HTTP/1.1 503 Service Unavailable\r\nContent-Type: text/plain; charset=utf-8\r\nContent-Length: %d\r\nConnection: close\r\n\r\n%s",
			len(body),
			body,
		)
		return
	}

	rawUpstream, err := net.DialTimeout(
		"tcp",
		fmt.Sprintf("127.0.0.1:%d", frontlineUpstreamPort),
		5*time.Second,
	)
	if err != nil {
		return
	}
	upstream := tls.Client(rawUpstream, p.tlsConfig)
	if err = upstream.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
		_ = upstream.Close()
		return
	}
	if err = upstream.Handshake(); err != nil {
		_ = upstream.Close()
		return
	}
	if err = upstream.SetDeadline(time.Time{}); err != nil {
		_ = upstream.Close()
		return
	}
	if _, err = upstream.Write(request); err != nil {
		_ = upstream.Close()
		return
	}

	bridgeConnections(client, upstream)
}

func (p *frontlineProxy) isPortalHostname(hostname string) bool {
	return hostname == p.publicHostname || e2bHostnamePattern.MatchString(hostname)
}

// readInitialRequest avoids net/http normalization before Frontline sees the
// request. In particular, it retains the exact Host and Upgrade headers and any
// request body bytes received with the header block.
func readInitialRequest(connection net.Conn) ([]byte, error) {
	request := make([]byte, 0, 4096)
	buffer := make([]byte, 4096)
	for !bytes.Contains(request, headerTerminator) && len(request) < bufferSize {
		readSize := min(len(buffer), bufferSize-len(request))
		n, err := connection.Read(buffer[:readSize])
		if n > 0 {
			request = append(request, buffer[:n]...)
		}
		if err != nil {
			if err == io.EOF {
				return request, nil
			}
			return nil, err
		}
	}
	return request, nil
}

func requestPath(request []byte) []byte {
	line, _, _ := bytes.Cut(request, []byte("\r\n"))
	fields := bytes.Fields(line)
	if len(fields) < 2 {
		return nil
	}
	path, _, _ := bytes.Cut(fields[1], []byte("?"))
	return path
}

func requestHostname(request []byte) string {
	lines := bytes.Split(request, []byte("\r\n"))
	for _, line := range lines[1:] {
		name, value, ok := bytes.Cut(line, []byte(":"))
		if !ok || !bytes.EqualFold(name, []byte("host")) {
			continue
		}
		host := strings.TrimSpace(string(value))
		if hostname, _, err := net.SplitHostPort(host); err == nil {
			return hostname
		}
		return strings.SplitN(host, ":", 2)[0]
	}
	return ""
}
