package main

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/cilium/ebpf"
	api "github.com/unkeyed/unkey/svc/heimdall/integration/ciliumplugin/datapathplugins"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const targetName = "cil_from_container"
const operationsRoot = "/sys/fs/bpf/cilium/plugins/heimdall"

type server struct {
	api.UnimplementedDatapathPluginServer
	namespace string
	spec      *ebpf.CollectionSpec
}

func (s *server) PrepareCollection(ctx context.Context, req *api.PrepareCollectionRequest) (*api.PrepareCollectionResponse, error) {
	if err := checkVersion(ctx); err != nil {
		return nil, err
	}
	lxc := req.GetAttachmentContext().GetLxc()
	if lxc == nil || lxc.GetPodInfo().GetNamespace() != s.namespace {
		return &api.PrepareCollectionResponse{Hooks: nil, Cookie: ""}, nil
	}
	program := req.GetCollection().GetPrograms()[targetName]
	if program == nil || ebpf.ProgramType(program.Type) != ebpf.SchedCLS || !strings.HasSuffix(program.SectionName, "/entry") {
		return nil, status.Error(codes.FailedPrecondition, "missing TC entrypoint")
	}
	if lxc.GetIface().GetName() == "" || lxc.GetPodInfo().GetName() == "" {
		return nil, status.Error(codes.InvalidArgument, "missing pod or interface name")
	}
	cookie := rand.Text()
	slog.Info("prepared", "generation", cookie, "declared_context", lxc, "identity", "unverified")
	return &api.PrepareCollectionResponse{Cookie: cookie, Hooks: []*api.PrepareCollectionResponse_HookSpec{
		{Type: api.HookType_PRE, Target: targetName, Constraints: nil},
		{Type: api.HookType_POST, Target: targetName, Constraints: nil},
	}}, nil
}

func (s *server) InstrumentCollection(ctx context.Context, req *api.InstrumentCollectionRequest) (*api.InstrumentCollectionResponse, error) {
	if err := checkVersion(ctx); err != nil {
		return nil, err
	}
	if err := s.validate(req); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if err := s.instrument(ctx, req); err != nil {
		slog.Error("instrument failed", "generation", req.Cookie, "error", err)
		return nil, status.Error(codes.Internal, err.Error())
	}
	slog.Info("programs supplied", "attachment", "unconfirmed", "identity", "unverified",
		"generation", req.Cookie, "declared_context", req.AttachmentContext, "hooks", req.Hooks, "pins", req.Pins)
	return &api.InstrumentCollectionResponse{}, nil
}

func checkVersion(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return status.FromContextError(err).Err()
	}
	md, ok := metadata.FromIncomingContext(ctx)
	versions := md.Get("cilium_version")
	if !ok || len(versions) != 1 {
		return status.Error(codes.FailedPrecondition, "requires Cilium version metadata")
	}
	fields := strings.Fields(versions[0])
	if len(fields) == 0 || fields[0] != "1.20.2" {
		return status.Error(codes.FailedPrecondition, "requires Cilium 1.20.2")
	}
	return nil
}

func (s *server) validate(req *api.InstrumentCollectionRequest) error {
	lxc := req.GetAttachmentContext().GetLxc()
	if lxc.GetPodInfo().GetNamespace() != s.namespace || lxc.GetPodInfo().GetName() == "" || lxc.GetIface().GetName() == "" {
		return errors.New("unexpected attachment context")
	}
	if req.Cookie == "" || len(req.Hooks) != 2 {
		return errors.New("expected generation and two hooks")
	}
	rel, err := filepath.Rel(operationsRoot, req.Pins)
	if err != nil || !filepath.IsLocal(rel) || filepath.Base(rel) != "pins" || len(strings.Split(rel, string(filepath.Separator))) != 2 {
		return errors.New("pins must be in a heimdall staging operation")
	}
	targetID := req.GetCollection().GetPrograms()[targetName].GetId()
	if targetID == 0 {
		return errors.New("missing target program ID")
	}
	seen := make(map[api.HookType]bool)
	for _, hook := range req.Hooks {
		if hook.GetType() != api.HookType_PRE && hook.GetType() != api.HookType_POST {
			return errors.New("unexpected hook type")
		}
		if seen[hook.Type] || hook.Target != targetName || hook.GetAttachTarget().GetProgramId() != targetID {
			return errors.New("duplicate hook or mismatched target")
		}
		seen[hook.Type] = true
		subprogram := "__pre_hook_heimdall__"
		if hook.Type == api.HookType_POST {
			subprogram = "__post_hook_heimdall__"
		}
		pin := filepath.Join(filepath.Dir(req.Pins), "hooks", targetName+"_"+subprogram)
		if hook.GetAttachTarget().GetSubprogName() != subprogram || hook.PinPath != pin {
			return errors.New("unexpected hook subprogram or pin path")
		}
	}
	return nil
}

func (s *server) instrument(ctx context.Context, req *api.InstrumentCollectionRequest) (err error) {
	spec := s.spec.Copy()
	target, err := ebpf.NewProgramFromID(ebpf.ProgramID(req.Hooks[0].AttachTarget.ProgramId))
	if err != nil {
		return fmt.Errorf("open target: %w", err)
	}
	defer func() { err = errors.Join(err, target.Close()) }()
	for _, hook := range req.Hooks {
		program := spec.Programs[hookProgram(hook.Type)]
		program.AttachTarget = target
		program.AttachTo = hook.AttachTarget.SubprogName
	}
	collection, err := ebpf.NewCollection(spec)
	if err != nil {
		return fmt.Errorf("load hooks: %w", err)
	}
	defer collection.Close()
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := collection.Maps["observations"].Pin(filepath.Join(req.Pins, "observations")); err != nil {
		return fmt.Errorf("pin observations: %w", err)
	}
	for _, hook := range req.Hooks {
		if err := collection.Programs[hookProgram(hook.Type)].Pin(hook.PinPath); err != nil {
			return fmt.Errorf("pin hook: %w", err)
		}
	}
	return nil
}

func hookProgram(kind api.HookType) string {
	if kind == api.HookType_PRE {
		return "before"
	}
	return "after"
}
