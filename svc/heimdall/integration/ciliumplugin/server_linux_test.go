package main

import (
	"context"
	"crypto/rand"
	"path/filepath"
	"testing"

	"github.com/cilium/ebpf"
	"github.com/stretchr/testify/require"
	api "github.com/unkeyed/unkey/svc/heimdall/integration/ciliumplugin/datapathplugins"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestPrepareCollectionLimitsInstrumentation(t *testing.T) {
	s := &server{namespace: "diagnostic"}
	for _, tc := range []struct {
		name      string
		version   string
		namespace string
		program   *api.PrepareCollectionRequest_CollectionSpec_ProgramSpec
		code      codes.Code
		hooks     int
	}{
		{"matching", "1.20.2 commit timestamp", "diagnostic", &api.PrepareCollectionRequest_CollectionSpec_ProgramSpec{Type: uint32(ebpf.SchedCLS), SectionName: "tc/entry"}, codes.OK, 2},
		{"outside namespace", "1.20.2", "kube-system", nil, codes.OK, 0},
		{"wrong version", "1.20.20", "diagnostic", nil, codes.FailedPrecondition, 0},
		{"no version", "", "diagnostic", nil, codes.FailedPrecondition, 0},
		{"missing entrypoint", "1.20.2", "diagnostic", nil, codes.FailedPrecondition, 0},
		{"tail program", "1.20.2", "diagnostic", &api.PrepareCollectionRequest_CollectionSpec_ProgramSpec{Type: uint32(ebpf.SchedCLS), SectionName: "tc/tail"}, codes.FailedPrecondition, 0},
		{"wrong program type", "1.20.2", "diagnostic", &api.PrepareCollectionRequest_CollectionSpec_ProgramSpec{Type: uint32(ebpf.XDP), SectionName: "xdp/entry"}, codes.FailedPrecondition, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := metadata.NewIncomingContext(t.Context(), metadata.Pairs("cilium_version", tc.version))
			req := &api.PrepareCollectionRequest{
				AttachmentContext: attachmentContext(t, tc.namespace),
				Collection: &api.PrepareCollectionRequest_CollectionSpec{
					Programs: map[string]*api.PrepareCollectionRequest_CollectionSpec_ProgramSpec{targetName: tc.program},
				},
			}
			resp, err := s.PrepareCollection(ctx, req)
			require.Equal(t, tc.code, status.Code(err))
			require.Len(t, resp.GetHooks(), tc.hooks)
			if tc.hooks > 0 {
				require.NotEmpty(t, resp.Cookie)
				require.Equal(t, api.HookType_PRE, resp.Hooks[0].Type)
				require.Equal(t, api.HookType_POST, resp.Hooks[1].Type)
				require.Equal(t, targetName, resp.Hooks[0].Target)
				require.Equal(t, targetName, resp.Hooks[1].Target)
			}
		})
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err := s.PrepareCollection(ctx, nil)
	require.Equal(t, codes.Canceled, status.Code(err))
}

func TestInstrumentCollectionRejectsInvalidHandoffBeforeLoading(t *testing.T) {
	s := &server{namespace: "diagnostic"}
	ctx := metadata.NewIncomingContext(t.Context(), metadata.Pairs("cilium_version", "1.20.2"))
	for _, tc := range []struct {
		name   string
		change func(*api.InstrumentCollectionRequest)
	}{
		{"outside namespace", func(r *api.InstrumentCollectionRequest) { r.AttachmentContext = attachmentContext(t, "kube-system") }},
		{"missing context", func(r *api.InstrumentCollectionRequest) { r.AttachmentContext = nil }},
		{"missing cookie", func(r *api.InstrumentCollectionRequest) { r.Cookie = "" }},
		{"missing hook", func(r *api.InstrumentCollectionRequest) { r.Hooks = r.Hooks[:1] }},
		{"nil hook", func(r *api.InstrumentCollectionRequest) { r.Hooks[0] = nil }},
		{"unknown hook", func(r *api.InstrumentCollectionRequest) { r.Hooks[0].Type = api.HookType_UNKNOWN }},
		{"duplicate hook", func(r *api.InstrumentCollectionRequest) { r.Hooks[1] = r.Hooks[0] }},
		{"different entrypoint", func(r *api.InstrumentCollectionRequest) { r.Hooks[0].Target = "cil_to_container" }},
		{"missing program ID", func(r *api.InstrumentCollectionRequest) { r.Collection = nil }},
		{"different target ID", func(r *api.InstrumentCollectionRequest) { r.Hooks[0].AttachTarget.ProgramId++ }},
		{"other subprogram", func(r *api.InstrumentCollectionRequest) { r.Hooks[0].AttachTarget.SubprogName = "original" }},
		{"outside pin root", func(r *api.InstrumentCollectionRequest) { r.Pins = "/sys/fs/bpf/heimdall/pins" }},
		{"traversal", func(r *api.InstrumentCollectionRequest) { r.Pins = operationsRoot + "/../../escape/pins" }},
		{"other plugin", func(r *api.InstrumentCollectionRequest) {
			r.Hooks[0].PinPath = "/sys/fs/bpf/cilium/plugins/other/hooks/pre"
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			operation := filepath.Join(operationsRoot, rand.Text())
			req := &api.InstrumentCollectionRequest{
				AttachmentContext: attachmentContext(t, "diagnostic"),
				Cookie:            rand.Text(),
				Pins:              filepath.Join(operation, "pins"),
				Collection: &api.InstrumentCollectionRequest_Collection{
					Programs: map[string]*api.InstrumentCollectionRequest_Collection_Program{targetName: {Id: 1}},
				},
			}
			for _, kind := range []api.HookType{api.HookType_PRE, api.HookType_POST} {
				subprogram := "__pre_hook_heimdall__"
				if kind == api.HookType_POST {
					subprogram = "__post_hook_heimdall__"
				}
				req.Hooks = append(req.Hooks, &api.InstrumentCollectionRequest_Hook{
					Type: kind, Target: targetName,
					AttachTarget: &api.InstrumentCollectionRequest_Hook_AttachTarget{ProgramId: 1, SubprogName: subprogram},
					PinPath:      filepath.Join(operation, "hooks", targetName+"_"+subprogram),
				})
			}
			tc.change(req)
			_, err := s.InstrumentCollection(ctx, req)
			require.Equal(t, codes.InvalidArgument, status.Code(err))
		})
	}
}

func attachmentContext(t *testing.T, namespace string) *api.AttachmentContext {
	t.Helper()
	return &api.AttachmentContext{Context: &api.AttachmentContext_Lxc{Lxc: &api.AttachmentContext_LXC{
		Iface:   &api.AttachmentContext_InterfaceInfo{Name: "lxc" + rand.Text()[:8]},
		PodInfo: &api.AttachmentContext_PodInfo{Namespace: namespace, Name: rand.Text()},
	}}}
}
