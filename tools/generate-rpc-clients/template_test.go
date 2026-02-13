package main

import (
	"bytes"
	"go/format"
	"strings"
	"testing"
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
				Name: "TestServiceClient",
				Methods: []methodInfo{
					{Name: "GetItem", ReqType: "GetItemRequest", RespType: "GetItemResponse", Kind: methodKindUnary},
				},
			},
		},
	}

	output := renderAndValidate(t, data)

	if !strings.Contains(output, "resp.Msg") {
		t.Error("unary method should contain resp.Msg unwrapping")
	}
	if !strings.Contains(output, "(*v1.GetItemResponse, error)") {
		t.Error("unary interface method should return plain proto response")
	}
	if strings.Contains(output, "ServerStreamForClient") {
		t.Error("unary-only output should not contain ServerStreamForClient")
	}
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
				Name: "WatchServiceClient",
				Methods: []methodInfo{
					{Name: "WatchEvents", ReqType: "WatchEventsRequest", RespType: "EventState", Kind: methodKindServerStream},
				},
			},
		},
	}

	output := renderAndValidate(t, data)

	if !strings.Contains(output, "ServerStreamForClient[v1.EventState]") {
		t.Error("streaming interface method should contain ServerStreamForClient in signature")
	}
	if !strings.Contains(output, "return c.inner.WatchEvents(ctx, connect.NewRequest(req))") {
		t.Error("streaming adapter should directly return the inner call")
	}
	if strings.Contains(output, "resp.Msg") {
		t.Error("streaming output should not contain resp.Msg unwrapping")
	}
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
				Name: "MixedServiceClient",
				Methods: []methodInfo{
					{Name: "WatchItems", ReqType: "WatchItemsRequest", RespType: "ItemState", Kind: methodKindServerStream},
					{Name: "GetItem", ReqType: "GetItemRequest", RespType: "GetItemResponse", Kind: methodKindUnary},
				},
			},
		},
	}

	output := renderAndValidate(t, data)

	if !strings.Contains(output, "ServerStreamForClient[v1.ItemState]") {
		t.Error("mixed output should contain ServerStreamForClient for streaming method")
	}
	if !strings.Contains(output, "resp.Msg") {
		t.Error("mixed output should contain resp.Msg for unary method")
	}
	if !strings.Contains(output, "return c.inner.WatchItems(ctx, connect.NewRequest(req))") {
		t.Error("streaming adapter should directly return the inner call")
	}
}
