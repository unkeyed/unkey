// Code generated by protoc-gen-connect-go. DO NOT EDIT.
//
// Source: proto/deploy/builderd/v1/builder.proto

package builderdv1connect

import (
	connect "connectrpc.com/connect"
	context "context"
	errors "errors"
	v1 "github.com/unkeyed/unkey/go/gen/proto/deploy/builderd/v1"
	http "net/http"
	strings "strings"
)

// This is a compile-time assertion to ensure that this generated file and the connect package are
// compatible. If you get a compiler error that this constant is not defined, this code was
// generated with a version of connect newer than the one compiled into your binary. You can fix the
// problem by either regenerating this code with an older version of connect or updating the connect
// version compiled into your binary.
const _ = connect.IsAtLeastVersion1_13_0

const (
	// BuilderServiceName is the fully-qualified name of the BuilderService service.
	BuilderServiceName = "deploy.builderd.v1.BuilderService"
)

// These constants are the fully-qualified names of the RPCs defined in this package. They're
// exposed at runtime as Spec.Procedure and as the final two segments of the HTTP route.
//
// Note that these are different from the fully-qualified method names used by
// google.golang.org/protobuf/reflect/protoreflect. To convert from these constants to
// reflection-formatted method names, remove the leading slash and convert the remaining slash to a
// period.
const (
	// BuilderServiceCreateBuildProcedure is the fully-qualified name of the BuilderService's
	// CreateBuild RPC.
	BuilderServiceCreateBuildProcedure = "/deploy.builderd.v1.BuilderService/CreateBuild"
	// BuilderServiceGetBuildProcedure is the fully-qualified name of the BuilderService's GetBuild RPC.
	BuilderServiceGetBuildProcedure = "/deploy.builderd.v1.BuilderService/GetBuild"
	// BuilderServiceListBuildsProcedure is the fully-qualified name of the BuilderService's ListBuilds
	// RPC.
	BuilderServiceListBuildsProcedure = "/deploy.builderd.v1.BuilderService/ListBuilds"
	// BuilderServiceCancelBuildProcedure is the fully-qualified name of the BuilderService's
	// CancelBuild RPC.
	BuilderServiceCancelBuildProcedure = "/deploy.builderd.v1.BuilderService/CancelBuild"
	// BuilderServiceDeleteBuildProcedure is the fully-qualified name of the BuilderService's
	// DeleteBuild RPC.
	BuilderServiceDeleteBuildProcedure = "/deploy.builderd.v1.BuilderService/DeleteBuild"
	// BuilderServiceStreamBuildLogsProcedure is the fully-qualified name of the BuilderService's
	// StreamBuildLogs RPC.
	BuilderServiceStreamBuildLogsProcedure = "/deploy.builderd.v1.BuilderService/StreamBuildLogs"
	// BuilderServiceGetTenantQuotasProcedure is the fully-qualified name of the BuilderService's
	// GetTenantQuotas RPC.
	BuilderServiceGetTenantQuotasProcedure = "/deploy.builderd.v1.BuilderService/GetTenantQuotas"
	// BuilderServiceGetBuildStatsProcedure is the fully-qualified name of the BuilderService's
	// GetBuildStats RPC.
	BuilderServiceGetBuildStatsProcedure = "/deploy.builderd.v1.BuilderService/GetBuildStats"
)

// These variables are the protoreflect.Descriptor objects for the RPCs defined in this package.
var (
	builderServiceServiceDescriptor               = v1.File_proto_deploy_builderd_v1_builder_proto.Services().ByName("BuilderService")
	builderServiceCreateBuildMethodDescriptor     = builderServiceServiceDescriptor.Methods().ByName("CreateBuild")
	builderServiceGetBuildMethodDescriptor        = builderServiceServiceDescriptor.Methods().ByName("GetBuild")
	builderServiceListBuildsMethodDescriptor      = builderServiceServiceDescriptor.Methods().ByName("ListBuilds")
	builderServiceCancelBuildMethodDescriptor     = builderServiceServiceDescriptor.Methods().ByName("CancelBuild")
	builderServiceDeleteBuildMethodDescriptor     = builderServiceServiceDescriptor.Methods().ByName("DeleteBuild")
	builderServiceStreamBuildLogsMethodDescriptor = builderServiceServiceDescriptor.Methods().ByName("StreamBuildLogs")
	builderServiceGetTenantQuotasMethodDescriptor = builderServiceServiceDescriptor.Methods().ByName("GetTenantQuotas")
	builderServiceGetBuildStatsMethodDescriptor   = builderServiceServiceDescriptor.Methods().ByName("GetBuildStats")
)

// BuilderServiceClient is a client for the deploy.builderd.v1.BuilderService service.
type BuilderServiceClient interface {
	// Create a new build job
	CreateBuild(context.Context, *connect.Request[v1.CreateBuildRequest]) (*connect.Response[v1.CreateBuildResponse], error)
	// Get build status and progress
	GetBuild(context.Context, *connect.Request[v1.GetBuildRequest]) (*connect.Response[v1.GetBuildResponse], error)
	// List builds with filtering (tenant-scoped)
	ListBuilds(context.Context, *connect.Request[v1.ListBuildsRequest]) (*connect.Response[v1.ListBuildsResponse], error)
	// Cancel a running build
	CancelBuild(context.Context, *connect.Request[v1.CancelBuildRequest]) (*connect.Response[v1.CancelBuildResponse], error)
	// Delete a build and its artifacts
	DeleteBuild(context.Context, *connect.Request[v1.DeleteBuildRequest]) (*connect.Response[v1.DeleteBuildResponse], error)
	// Stream build logs in real-time
	StreamBuildLogs(context.Context, *connect.Request[v1.StreamBuildLogsRequest]) (*connect.ServerStreamForClient[v1.StreamBuildLogsResponse], error)
	// Get tenant quotas and usage
	GetTenantQuotas(context.Context, *connect.Request[v1.GetTenantQuotasRequest]) (*connect.Response[v1.GetTenantQuotasResponse], error)
	// Get build statistics
	GetBuildStats(context.Context, *connect.Request[v1.GetBuildStatsRequest]) (*connect.Response[v1.GetBuildStatsResponse], error)
}

// NewBuilderServiceClient constructs a client for the deploy.builderd.v1.BuilderService service. By
// default, it uses the Connect protocol with the binary Protobuf Codec, asks for gzipped responses,
// and sends uncompressed requests. To use the gRPC or gRPC-Web protocols, supply the
// connect.WithGRPC() or connect.WithGRPCWeb() options.
//
// The URL supplied here should be the base URL for the Connect or gRPC server (for example,
// http://api.acme.com or https://acme.com/grpc).
func NewBuilderServiceClient(httpClient connect.HTTPClient, baseURL string, opts ...connect.ClientOption) BuilderServiceClient {
	baseURL = strings.TrimRight(baseURL, "/")
	return &builderServiceClient{
		createBuild: connect.NewClient[v1.CreateBuildRequest, v1.CreateBuildResponse](
			httpClient,
			baseURL+BuilderServiceCreateBuildProcedure,
			connect.WithSchema(builderServiceCreateBuildMethodDescriptor),
			connect.WithClientOptions(opts...),
		),
		getBuild: connect.NewClient[v1.GetBuildRequest, v1.GetBuildResponse](
			httpClient,
			baseURL+BuilderServiceGetBuildProcedure,
			connect.WithSchema(builderServiceGetBuildMethodDescriptor),
			connect.WithClientOptions(opts...),
		),
		listBuilds: connect.NewClient[v1.ListBuildsRequest, v1.ListBuildsResponse](
			httpClient,
			baseURL+BuilderServiceListBuildsProcedure,
			connect.WithSchema(builderServiceListBuildsMethodDescriptor),
			connect.WithClientOptions(opts...),
		),
		cancelBuild: connect.NewClient[v1.CancelBuildRequest, v1.CancelBuildResponse](
			httpClient,
			baseURL+BuilderServiceCancelBuildProcedure,
			connect.WithSchema(builderServiceCancelBuildMethodDescriptor),
			connect.WithClientOptions(opts...),
		),
		deleteBuild: connect.NewClient[v1.DeleteBuildRequest, v1.DeleteBuildResponse](
			httpClient,
			baseURL+BuilderServiceDeleteBuildProcedure,
			connect.WithSchema(builderServiceDeleteBuildMethodDescriptor),
			connect.WithClientOptions(opts...),
		),
		streamBuildLogs: connect.NewClient[v1.StreamBuildLogsRequest, v1.StreamBuildLogsResponse](
			httpClient,
			baseURL+BuilderServiceStreamBuildLogsProcedure,
			connect.WithSchema(builderServiceStreamBuildLogsMethodDescriptor),
			connect.WithClientOptions(opts...),
		),
		getTenantQuotas: connect.NewClient[v1.GetTenantQuotasRequest, v1.GetTenantQuotasResponse](
			httpClient,
			baseURL+BuilderServiceGetTenantQuotasProcedure,
			connect.WithSchema(builderServiceGetTenantQuotasMethodDescriptor),
			connect.WithClientOptions(opts...),
		),
		getBuildStats: connect.NewClient[v1.GetBuildStatsRequest, v1.GetBuildStatsResponse](
			httpClient,
			baseURL+BuilderServiceGetBuildStatsProcedure,
			connect.WithSchema(builderServiceGetBuildStatsMethodDescriptor),
			connect.WithClientOptions(opts...),
		),
	}
}

// builderServiceClient implements BuilderServiceClient.
type builderServiceClient struct {
	createBuild     *connect.Client[v1.CreateBuildRequest, v1.CreateBuildResponse]
	getBuild        *connect.Client[v1.GetBuildRequest, v1.GetBuildResponse]
	listBuilds      *connect.Client[v1.ListBuildsRequest, v1.ListBuildsResponse]
	cancelBuild     *connect.Client[v1.CancelBuildRequest, v1.CancelBuildResponse]
	deleteBuild     *connect.Client[v1.DeleteBuildRequest, v1.DeleteBuildResponse]
	streamBuildLogs *connect.Client[v1.StreamBuildLogsRequest, v1.StreamBuildLogsResponse]
	getTenantQuotas *connect.Client[v1.GetTenantQuotasRequest, v1.GetTenantQuotasResponse]
	getBuildStats   *connect.Client[v1.GetBuildStatsRequest, v1.GetBuildStatsResponse]
}

// CreateBuild calls deploy.builderd.v1.BuilderService.CreateBuild.
func (c *builderServiceClient) CreateBuild(ctx context.Context, req *connect.Request[v1.CreateBuildRequest]) (*connect.Response[v1.CreateBuildResponse], error) {
	return c.createBuild.CallUnary(ctx, req)
}

// GetBuild calls deploy.builderd.v1.BuilderService.GetBuild.
func (c *builderServiceClient) GetBuild(ctx context.Context, req *connect.Request[v1.GetBuildRequest]) (*connect.Response[v1.GetBuildResponse], error) {
	return c.getBuild.CallUnary(ctx, req)
}

// ListBuilds calls deploy.builderd.v1.BuilderService.ListBuilds.
func (c *builderServiceClient) ListBuilds(ctx context.Context, req *connect.Request[v1.ListBuildsRequest]) (*connect.Response[v1.ListBuildsResponse], error) {
	return c.listBuilds.CallUnary(ctx, req)
}

// CancelBuild calls deploy.builderd.v1.BuilderService.CancelBuild.
func (c *builderServiceClient) CancelBuild(ctx context.Context, req *connect.Request[v1.CancelBuildRequest]) (*connect.Response[v1.CancelBuildResponse], error) {
	return c.cancelBuild.CallUnary(ctx, req)
}

// DeleteBuild calls deploy.builderd.v1.BuilderService.DeleteBuild.
func (c *builderServiceClient) DeleteBuild(ctx context.Context, req *connect.Request[v1.DeleteBuildRequest]) (*connect.Response[v1.DeleteBuildResponse], error) {
	return c.deleteBuild.CallUnary(ctx, req)
}

// StreamBuildLogs calls deploy.builderd.v1.BuilderService.StreamBuildLogs.
func (c *builderServiceClient) StreamBuildLogs(ctx context.Context, req *connect.Request[v1.StreamBuildLogsRequest]) (*connect.ServerStreamForClient[v1.StreamBuildLogsResponse], error) {
	return c.streamBuildLogs.CallServerStream(ctx, req)
}

// GetTenantQuotas calls deploy.builderd.v1.BuilderService.GetTenantQuotas.
func (c *builderServiceClient) GetTenantQuotas(ctx context.Context, req *connect.Request[v1.GetTenantQuotasRequest]) (*connect.Response[v1.GetTenantQuotasResponse], error) {
	return c.getTenantQuotas.CallUnary(ctx, req)
}

// GetBuildStats calls deploy.builderd.v1.BuilderService.GetBuildStats.
func (c *builderServiceClient) GetBuildStats(ctx context.Context, req *connect.Request[v1.GetBuildStatsRequest]) (*connect.Response[v1.GetBuildStatsResponse], error) {
	return c.getBuildStats.CallUnary(ctx, req)
}

// BuilderServiceHandler is an implementation of the deploy.builderd.v1.BuilderService service.
type BuilderServiceHandler interface {
	// Create a new build job
	CreateBuild(context.Context, *connect.Request[v1.CreateBuildRequest]) (*connect.Response[v1.CreateBuildResponse], error)
	// Get build status and progress
	GetBuild(context.Context, *connect.Request[v1.GetBuildRequest]) (*connect.Response[v1.GetBuildResponse], error)
	// List builds with filtering (tenant-scoped)
	ListBuilds(context.Context, *connect.Request[v1.ListBuildsRequest]) (*connect.Response[v1.ListBuildsResponse], error)
	// Cancel a running build
	CancelBuild(context.Context, *connect.Request[v1.CancelBuildRequest]) (*connect.Response[v1.CancelBuildResponse], error)
	// Delete a build and its artifacts
	DeleteBuild(context.Context, *connect.Request[v1.DeleteBuildRequest]) (*connect.Response[v1.DeleteBuildResponse], error)
	// Stream build logs in real-time
	StreamBuildLogs(context.Context, *connect.Request[v1.StreamBuildLogsRequest], *connect.ServerStream[v1.StreamBuildLogsResponse]) error
	// Get tenant quotas and usage
	GetTenantQuotas(context.Context, *connect.Request[v1.GetTenantQuotasRequest]) (*connect.Response[v1.GetTenantQuotasResponse], error)
	// Get build statistics
	GetBuildStats(context.Context, *connect.Request[v1.GetBuildStatsRequest]) (*connect.Response[v1.GetBuildStatsResponse], error)
}

// NewBuilderServiceHandler builds an HTTP handler from the service implementation. It returns the
// path on which to mount the handler and the handler itself.
//
// By default, handlers support the Connect, gRPC, and gRPC-Web protocols with the binary Protobuf
// and JSON codecs. They also support gzip compression.
func NewBuilderServiceHandler(svc BuilderServiceHandler, opts ...connect.HandlerOption) (string, http.Handler) {
	builderServiceCreateBuildHandler := connect.NewUnaryHandler(
		BuilderServiceCreateBuildProcedure,
		svc.CreateBuild,
		connect.WithSchema(builderServiceCreateBuildMethodDescriptor),
		connect.WithHandlerOptions(opts...),
	)
	builderServiceGetBuildHandler := connect.NewUnaryHandler(
		BuilderServiceGetBuildProcedure,
		svc.GetBuild,
		connect.WithSchema(builderServiceGetBuildMethodDescriptor),
		connect.WithHandlerOptions(opts...),
	)
	builderServiceListBuildsHandler := connect.NewUnaryHandler(
		BuilderServiceListBuildsProcedure,
		svc.ListBuilds,
		connect.WithSchema(builderServiceListBuildsMethodDescriptor),
		connect.WithHandlerOptions(opts...),
	)
	builderServiceCancelBuildHandler := connect.NewUnaryHandler(
		BuilderServiceCancelBuildProcedure,
		svc.CancelBuild,
		connect.WithSchema(builderServiceCancelBuildMethodDescriptor),
		connect.WithHandlerOptions(opts...),
	)
	builderServiceDeleteBuildHandler := connect.NewUnaryHandler(
		BuilderServiceDeleteBuildProcedure,
		svc.DeleteBuild,
		connect.WithSchema(builderServiceDeleteBuildMethodDescriptor),
		connect.WithHandlerOptions(opts...),
	)
	builderServiceStreamBuildLogsHandler := connect.NewServerStreamHandler(
		BuilderServiceStreamBuildLogsProcedure,
		svc.StreamBuildLogs,
		connect.WithSchema(builderServiceStreamBuildLogsMethodDescriptor),
		connect.WithHandlerOptions(opts...),
	)
	builderServiceGetTenantQuotasHandler := connect.NewUnaryHandler(
		BuilderServiceGetTenantQuotasProcedure,
		svc.GetTenantQuotas,
		connect.WithSchema(builderServiceGetTenantQuotasMethodDescriptor),
		connect.WithHandlerOptions(opts...),
	)
	builderServiceGetBuildStatsHandler := connect.NewUnaryHandler(
		BuilderServiceGetBuildStatsProcedure,
		svc.GetBuildStats,
		connect.WithSchema(builderServiceGetBuildStatsMethodDescriptor),
		connect.WithHandlerOptions(opts...),
	)
	return "/deploy.builderd.v1.BuilderService/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case BuilderServiceCreateBuildProcedure:
			builderServiceCreateBuildHandler.ServeHTTP(w, r)
		case BuilderServiceGetBuildProcedure:
			builderServiceGetBuildHandler.ServeHTTP(w, r)
		case BuilderServiceListBuildsProcedure:
			builderServiceListBuildsHandler.ServeHTTP(w, r)
		case BuilderServiceCancelBuildProcedure:
			builderServiceCancelBuildHandler.ServeHTTP(w, r)
		case BuilderServiceDeleteBuildProcedure:
			builderServiceDeleteBuildHandler.ServeHTTP(w, r)
		case BuilderServiceStreamBuildLogsProcedure:
			builderServiceStreamBuildLogsHandler.ServeHTTP(w, r)
		case BuilderServiceGetTenantQuotasProcedure:
			builderServiceGetTenantQuotasHandler.ServeHTTP(w, r)
		case BuilderServiceGetBuildStatsProcedure:
			builderServiceGetBuildStatsHandler.ServeHTTP(w, r)
		default:
			http.NotFound(w, r)
		}
	})
}

// UnimplementedBuilderServiceHandler returns CodeUnimplemented from all methods.
type UnimplementedBuilderServiceHandler struct{}

func (UnimplementedBuilderServiceHandler) CreateBuild(context.Context, *connect.Request[v1.CreateBuildRequest]) (*connect.Response[v1.CreateBuildResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("deploy.builderd.v1.BuilderService.CreateBuild is not implemented"))
}

func (UnimplementedBuilderServiceHandler) GetBuild(context.Context, *connect.Request[v1.GetBuildRequest]) (*connect.Response[v1.GetBuildResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("deploy.builderd.v1.BuilderService.GetBuild is not implemented"))
}

func (UnimplementedBuilderServiceHandler) ListBuilds(context.Context, *connect.Request[v1.ListBuildsRequest]) (*connect.Response[v1.ListBuildsResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("deploy.builderd.v1.BuilderService.ListBuilds is not implemented"))
}

func (UnimplementedBuilderServiceHandler) CancelBuild(context.Context, *connect.Request[v1.CancelBuildRequest]) (*connect.Response[v1.CancelBuildResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("deploy.builderd.v1.BuilderService.CancelBuild is not implemented"))
}

func (UnimplementedBuilderServiceHandler) DeleteBuild(context.Context, *connect.Request[v1.DeleteBuildRequest]) (*connect.Response[v1.DeleteBuildResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("deploy.builderd.v1.BuilderService.DeleteBuild is not implemented"))
}

func (UnimplementedBuilderServiceHandler) StreamBuildLogs(context.Context, *connect.Request[v1.StreamBuildLogsRequest], *connect.ServerStream[v1.StreamBuildLogsResponse]) error {
	return connect.NewError(connect.CodeUnimplemented, errors.New("deploy.builderd.v1.BuilderService.StreamBuildLogs is not implemented"))
}

func (UnimplementedBuilderServiceHandler) GetTenantQuotas(context.Context, *connect.Request[v1.GetTenantQuotasRequest]) (*connect.Response[v1.GetTenantQuotasResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("deploy.builderd.v1.BuilderService.GetTenantQuotas is not implemented"))
}

func (UnimplementedBuilderServiceHandler) GetBuildStats(context.Context, *connect.Request[v1.GetBuildStatsRequest]) (*connect.Response[v1.GetBuildStatsResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("deploy.builderd.v1.BuilderService.GetBuildStats is not implemented"))
}
