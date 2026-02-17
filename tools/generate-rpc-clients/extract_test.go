package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"github.com/stretchr/testify/require"
)

func parseSource(t *testing.T, src string) *ast.File {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("failed to parse source: %v", err)
	}
	return f
}

func TestFindProtoImport(t *testing.T) {
	cases := []struct {
		name, src, wantAlias, wantPath string
	}{
		{"named", `package fooconnect
import (
	v1 "github.com/example/gen/proto/foo/v1"
	"connectrpc.com/connect"
)
`, "v1", "github.com/example/gen/proto/foo/v1"},
		{"unnamed", `package barconnect
import (
	"github.com/example/gen/proto/bar/v1"
	"connectrpc.com/connect"
)
`, "v1", "github.com/example/gen/proto/bar/v1"},
		{"skips_well_known_proto", `package fooconnect
import (
	v1 "github.com/example/gen/proto/foo/v1"
	"google.golang.org/protobuf/types/known/emptypb"
	"connectrpc.com/connect"
)
`, "v1", "github.com/example/gen/proto/foo/v1"},
		{"no_match", `package fooconnect
import (
	"connectrpc.com/connect"
	"net/http"
)
`, "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := parseSource(t, tc.src)
			alias, path := findProtoImport(f)
			require.Equal(t, tc.wantAlias, alias)
			require.Equal(t, tc.wantPath, path)
		})
	}
}

func TestFindServices_UnaryOnly(t *testing.T) {
	src := `package testconnect

import (
	v1 "github.com/example/gen/proto/test/v1"
	"connectrpc.com/connect"
)

type TestServiceClient interface {
	GetItem(context.Context, *connect.Request[v1.GetItemRequest]) (*connect.Response[v1.GetItemResponse], error)
	CreateItem(context.Context, *connect.Request[v1.CreateItemRequest]) (*connect.Response[v1.CreateItemResponse], error)
}
`
	f := parseSource(t, src)
	services := findServices(f, "v1")

	if len(services) != 1 {
		t.Fatalf("got %d services, want 1", len(services))
	}

	svc := services[0]
	if svc.Name != "TestServiceClient" {
		t.Errorf("service name = %q, want %q", svc.Name, "TestServiceClient")
	}

	if len(svc.Methods) != 2 {
		t.Fatalf("got %d methods, want 2", len(svc.Methods))
	}

	for _, m := range svc.Methods {
		if m.Kind != methodKindUnary {
			t.Errorf("method %s kind = %q, want %q", m.Name, m.Kind, methodKindUnary)
		}
	}
}

func TestFindServices_ServerStream(t *testing.T) {
	src := `package testconnect

import (
	v1 "github.com/example/gen/proto/test/v1"
	"connectrpc.com/connect"
)

type WatchServiceClient interface {
	WatchEvents(context.Context, *connect.Request[v1.WatchEventsRequest]) (*connect.ServerStreamForClient[v1.EventState], error)
}
`
	f := parseSource(t, src)
	services := findServices(f, "v1")

	if len(services) != 1 {
		t.Fatalf("got %d services, want 1", len(services))
	}

	svc := services[0]
	if len(svc.Methods) != 1 {
		t.Fatalf("got %d methods, want 1", len(svc.Methods))
	}

	m := svc.Methods[0]
	if m.Name != "WatchEvents" {
		t.Errorf("method name = %q, want %q", m.Name, "WatchEvents")
	}
	if m.Kind != methodKindServerStream {
		t.Errorf("method kind = %q, want %q", m.Kind, methodKindServerStream)
	}
	if m.ReqType != "WatchEventsRequest" {
		t.Errorf("req type = %q, want %q", m.ReqType, "WatchEventsRequest")
	}
	if m.RespType != "EventState" {
		t.Errorf("resp type = %q, want %q", m.RespType, "EventState")
	}
}

func TestFindServices_Mixed(t *testing.T) {
	src := `package testconnect

import (
	v1 "github.com/example/gen/proto/test/v1"
	"connectrpc.com/connect"
)

type MixedServiceClient interface {
	WatchItems(context.Context, *connect.Request[v1.WatchItemsRequest]) (*connect.ServerStreamForClient[v1.ItemState], error)
	GetItem(context.Context, *connect.Request[v1.GetItemRequest]) (*connect.Response[v1.GetItemResponse], error)
	WatchLogs(context.Context, *connect.Request[v1.WatchLogsRequest]) (*connect.ServerStreamForClient[v1.LogEntry], error)
}
`
	f := parseSource(t, src)
	services := findServices(f, "v1")

	if len(services) != 1 {
		t.Fatalf("got %d services, want 1", len(services))
	}

	methods := services[0].Methods
	if len(methods) != 3 {
		t.Fatalf("got %d methods, want 3", len(methods))
	}

	expected := []struct {
		name string
		kind methodKind
	}{
		{"WatchItems", methodKindServerStream},
		{"GetItem", methodKindUnary},
		{"WatchLogs", methodKindServerStream},
	}

	for i, exp := range expected {
		if methods[i].Name != exp.name {
			t.Errorf("method[%d] name = %q, want %q", i, methods[i].Name, exp.name)
		}
		if methods[i].Kind != exp.kind {
			t.Errorf("method[%d] kind = %q, want %q", i, methods[i].Kind, exp.kind)
		}
	}
}

func TestFindServices_SkipsNonServiceClient(t *testing.T) {
	src := `package testconnect

import (
	v1 "github.com/example/gen/proto/test/v1"
	"connectrpc.com/connect"
)

type NotAClient interface {
	GetItem(context.Context, *connect.Request[v1.GetItemRequest]) (*connect.Response[v1.GetItemResponse], error)
}

type SomeHandler interface {
	Handle(context.Context, *connect.Request[v1.HandleRequest]) (*connect.Response[v1.HandleResponse], error)
}
`
	f := parseSource(t, src)
	services := findServices(f, "v1")

	if len(services) != 0 {
		t.Errorf("got %d services, want 0 (non-ServiceClient interfaces should be skipped)", len(services))
	}
}
