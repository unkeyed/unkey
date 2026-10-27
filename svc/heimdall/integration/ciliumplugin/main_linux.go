package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/link"
	api "github.com/unkeyed/unkey/svc/heimdall/integration/ciliumplugin/datapathplugins"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	socket := flag.String("socket", "/var/run/cilium/plugins/heimdall/plugin.sock", "plugin socket")
	namespace := flag.String("namespace", "", "required namespace allowlist; no billing attribution")
	dump := flag.String("dump", "", "read one generation's pinned observations")
	iface := flag.String("interface", "", "query actual ingress TCX attachment")
	endpoint := flag.String("inspect", "", "verify live hooks and read one Cilium endpoint directory")
	flag.Parse()
	if *endpoint != "" {
		return inspect(*endpoint)
	}
	if *dump != "" {
		return dumpObservations(*dump)
	}
	if *iface != "" {
		return dumpAttachment(*iface)
	}
	if *namespace == "" {
		return errors.New("namespace is required")
	}
	spec, err := loadBpf()
	if err != nil {
		return fmt.Errorf("load BPF specification: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(*socket), 0700); err != nil {
		return err
	}
	listener, err := net.Listen("unix", *socket)
	if err != nil {
		return err
	}
	defer func() {
		if err := listener.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
			slog.Error("close listener", "error", err)
		}
	}()
	if err := os.Chmod(*socket, 0600); err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	rpc := grpc.NewServer(grpc.MaxConcurrentStreams(4), grpc.UnaryInterceptor(admit(make(chan struct{}, 4))))
	api.RegisterDatapathPluginServer(rpc, &server{
		UnimplementedDatapathPluginServer: api.UnimplementedDatapathPluginServer{},
		namespace:                         *namespace,
		spec:                              spec,
	})
	finished := make(chan struct{})
	stopped := make(chan struct{})
	defer func() { close(finished); <-stopped }()
	go func() {
		defer close(stopped)
		select {
		case <-ctx.Done():
			rpc.Stop()
		case <-finished:
		}
	}()
	slog.Info("listening", "socket", *socket, "namespace", *namespace, "billing", "disabled")
	if err := rpc.Serve(listener); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
		return err
	}
	return nil
}

func admit(slots chan struct{}) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		select {
		case slots <- struct{}{}:
			defer func() { <-slots }()
		default:
			return nil, status.Error(codes.ResourceExhausted, "plugin busy")
		}
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		return handler(ctx, req)
	}
}

func dumpObservations(path string) (err error) {
	m, err := ebpf.LoadPinnedMap(path, nil)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, m.Close()) }()
	var values bpfObservation
	if err := m.Lookup(uint32(0), &values); err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(values)
}

func dumpAttachment(name string) error {
	device, err := net.InterfaceByName(name)
	if err != nil {
		return err
	}
	result, err := link.QueryPrograms(link.QueryOptions{Target: device.Index, Attach: ebpf.AttachTCXIngress, QueryFlags: 0})
	if err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(result)
}
