package main

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"net"
	"time"
)

// runHostFrontline exposes one deployment on a localhost port. Cursor has no
// Amp portal hostname, so the proxy restores the *.unkey.local Host header
// Frontline uses for route lookup and then TLS-dials the local Frontline.
func runHostFrontline(args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("usage: amp-orb-services host-frontline <listen-port> <source-hostname>")
	}
	listenPort, err := parsePort(args[0])
	if err != nil {
		return err
	}
	sourceHostname := args[1]
	if !hostnamePattern.MatchString(sourceHostname) {
		return fmt.Errorf("source route must be a valid hostname")
	}

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", listenPort))
	if err != nil {
		return err
	}
	defer func() { _ = listener.Close() }()

	tlsConfig := &tls.Config{ //nolint:exhaustruct,gosec
		MinVersion:         tls.VersionTLS12,
		ServerName:         sourceHostname,
		InsecureSkipVerify: true,
	}

	for {
		client, acceptErr := listener.Accept()
		if acceptErr != nil {
			fmt.Printf("Unable to accept app connection: %s\n", acceptErr)
			time.Sleep(100 * time.Millisecond)
			continue
		}
		go handleHostFrontline(client, sourceHostname, tlsConfig)
	}
}

func handleHostFrontline(client net.Conn, sourceHostname string, tlsConfig *tls.Config) {
	defer func() { _ = client.Close() }()

	request, err := readInitialRequest(client)
	if err != nil || len(request) == 0 {
		return
	}
	request = rewriteHostHeader(request, sourceHostname)

	rawUpstream, err := net.DialTimeout(
		"tcp",
		fmt.Sprintf("127.0.0.1:%d", frontlineUpstreamPort),
		5*time.Second,
	)
	if err != nil {
		body := []byte("Waiting for Frontline on localhost:9443\n")
		_, _ = fmt.Fprintf(
			client,
			"HTTP/1.1 503 Service Unavailable\r\nContent-Type: text/plain; charset=utf-8\r\nContent-Length: %d\r\nConnection: close\r\n\r\n%s",
			len(body),
			body,
		)
		return
	}
	upstream := tls.Client(rawUpstream, tlsConfig)
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

func rewriteHostHeader(request []byte, hostname string) []byte {
	lines := bytes.Split(request, []byte("\r\n"))
	hostLine := []byte("Host: " + hostname)
	replaced := false
	for i, line := range lines {
		name, _, ok := bytes.Cut(line, []byte(":"))
		if !ok || !bytes.EqualFold(name, []byte("host")) {
			continue
		}
		lines[i] = hostLine
		replaced = true
		break
	}
	if !replaced && len(lines) > 0 {
		inserted := make([][]byte, 0, len(lines)+1)
		inserted = append(inserted, lines[0], hostLine)
		inserted = append(inserted, lines[1:]...)
		lines = inserted
	}
	return bytes.Join(lines, []byte("\r\n"))
}
