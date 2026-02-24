package main

import (
	"bytes"
	"go/format"
	"testing"

	"github.com/stretchr/testify/require"
)

func renderAndValidate(t *testing.T, data fileData) string {
	t.Helper()
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		t.Fatalf("template execution failed: %v", err)
	}
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		t.Fatalf("generated code is not valid Go: %v\n%s", err, buf.String())
	}
	return string(formatted)
}

func TestTemplateOutput_Unary(t *testing.T) {
	data := fileData{
		PackageName:   "testrpc",
		ConnectPkg:    "testv1connect",
		ConnectImport: "github.com/example/gen/proto/test/v1/testv1connect",
		ProtoAlias:    "v1",
		ProtoImport:   "github.com/example/gen/proto/test/v1",
		Services: []serviceInfo{
			{
				Name:            "TestServiceClient",
				ServiceBaseName: "TestService",
				Methods: []methodInfo{
					{Name: "GetItem", ReqType: "GetItemRequest", RespType: "GetItemResponse", Kind: methodKindUnary},
				},
			},
		},
	}

	t.Run("unwraps response via resp.Msg", func(t *testing.T) {
		output := renderAndValidate(t, data)
		require.Contains(t, output, "resp.Msg")
	})
	t.Run("returns plain proto response type", func(t *testing.T) {
		output := renderAndValidate(t, data)
		require.Contains(t, output, "(*v1.GetItemResponse, error)")
	})
	t.Run("does not contain ServerStreamForClient", func(t *testing.T) {
		output := renderAndValidate(t, data)
		require.NotContains(t, output, "ServerStreamForClient")
	})
	t.Run("starts a tracing span", func(t *testing.T) {
		output := renderAndValidate(t, data)
		require.Contains(t, output, `tracing.Start(ctx, "TestService.GetItem")`)
		require.Contains(t, output, "defer span.End()")
	})
	t.Run("records error on span but skips not-found", func(t *testing.T) {
		output := renderAndValidate(t, data)
		require.Contains(t, output, "tracing.RecordError(span, err)")
		require.Contains(t, output, "connect.CodeOf(err) != connect.CodeNotFound")
	})
}

func TestTemplateOutput_ServerStream(t *testing.T) {
	data := fileData{
		PackageName:   "testrpc",
		ConnectPkg:    "testv1connect",
		ConnectImport: "github.com/example/gen/proto/test/v1/testv1connect",
		ProtoAlias:    "v1",
		ProtoImport:   "github.com/example/gen/proto/test/v1",
		Services: []serviceInfo{
			{
				Name:            "WatchServiceClient",
				ServiceBaseName: "WatchService",
				Methods: []methodInfo{
					{Name: "WatchEvents", ReqType: "WatchEventsRequest", RespType: "EventState", Kind: methodKindServerStream},
				},
			},
		},
	}

	t.Run("signature contains ServerStreamForClient", func(t *testing.T) {
		output := renderAndValidate(t, data)
		require.Contains(t, output, "ServerStreamForClient[v1.EventState]")
	})
	t.Run("adapter directly returns inner call", func(t *testing.T) {
		output := renderAndValidate(t, data)
		require.Contains(t, output, "return c.inner.WatchEvents(ctx, connect.NewRequest(req))")
	})
	t.Run("does not contain resp.Msg unwrapping", func(t *testing.T) {
		output := renderAndValidate(t, data)
		require.NotContains(t, output, "resp.Msg")
	})
	t.Run("does not contain tracing span", func(t *testing.T) {
		output := renderAndValidate(t, data)
		require.NotContains(t, output, "tracing.Start")
	})
}

func TestTemplateOutput_Mixed(t *testing.T) {
	data := fileData{
		PackageName:   "testrpc",
		ConnectPkg:    "testv1connect",
		ConnectImport: "github.com/example/gen/proto/test/v1/testv1connect",
		ProtoAlias:    "v1",
		ProtoImport:   "github.com/example/gen/proto/test/v1",
		Services: []serviceInfo{
			{
				Name:            "MixedServiceClient",
				ServiceBaseName: "MixedService",
				Methods: []methodInfo{
					{Name: "WatchItems", ReqType: "WatchItemsRequest", RespType: "ItemState", Kind: methodKindServerStream},
					{Name: "GetItem", ReqType: "GetItemRequest", RespType: "GetItemResponse", Kind: methodKindUnary},
				},
			},
		},
	}

	t.Run("contains ServerStreamForClient for streaming method", func(t *testing.T) {
		output := renderAndValidate(t, data)
		require.Contains(t, output, "ServerStreamForClient[v1.ItemState]")
	})
	t.Run("contains resp.Msg for unary method", func(t *testing.T) {
		output := renderAndValidate(t, data)
		require.Contains(t, output, "resp.Msg")
	})
	t.Run("streaming adapter directly returns inner call", func(t *testing.T) {
		output := renderAndValidate(t, data)
		require.Contains(t, output, "return c.inner.WatchItems(ctx, connect.NewRequest(req))")
	})
	t.Run("tracing span only on unary method", func(t *testing.T) {
		output := renderAndValidate(t, data)
		require.Contains(t, output, `tracing.Start(ctx, "MixedService.GetItem")`)
		require.NotContains(t, output, `tracing.Start(ctx, "MixedService.WatchItems")`)
	})
}
