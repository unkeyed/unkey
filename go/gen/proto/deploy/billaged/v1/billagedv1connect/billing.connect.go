// Code generated by protoc-gen-connect-go. DO NOT EDIT.
//
// Source: proto/deploy/billaged/v1/billing.proto

package billagedv1connect

import (
	connect "connectrpc.com/connect"
	context "context"
	errors "errors"
	v1 "github.com/unkeyed/unkey/go/gen/proto/deploy/billaged/v1"
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
	// BillingServiceName is the fully-qualified name of the BillingService service.
	BillingServiceName = "deploy.billaged.v1.BillingService"
)

// These constants are the fully-qualified names of the RPCs defined in this package. They're
// exposed at runtime as Spec.Procedure and as the final two segments of the HTTP route.
//
// Note that these are different from the fully-qualified method names used by
// google.golang.org/protobuf/reflect/protoreflect. To convert from these constants to
// reflection-formatted method names, remove the leading slash and convert the remaining slash to a
// period.
const (
	// BillingServiceSendMetricsBatchProcedure is the fully-qualified name of the BillingService's
	// SendMetricsBatch RPC.
	BillingServiceSendMetricsBatchProcedure = "/deploy.billaged.v1.BillingService/SendMetricsBatch"
	// BillingServiceSendHeartbeatProcedure is the fully-qualified name of the BillingService's
	// SendHeartbeat RPC.
	BillingServiceSendHeartbeatProcedure = "/deploy.billaged.v1.BillingService/SendHeartbeat"
	// BillingServiceNotifyVmStartedProcedure is the fully-qualified name of the BillingService's
	// NotifyVmStarted RPC.
	BillingServiceNotifyVmStartedProcedure = "/deploy.billaged.v1.BillingService/NotifyVmStarted"
	// BillingServiceNotifyVmStoppedProcedure is the fully-qualified name of the BillingService's
	// NotifyVmStopped RPC.
	BillingServiceNotifyVmStoppedProcedure = "/deploy.billaged.v1.BillingService/NotifyVmStopped"
	// BillingServiceNotifyPossibleGapProcedure is the fully-qualified name of the BillingService's
	// NotifyPossibleGap RPC.
	BillingServiceNotifyPossibleGapProcedure = "/deploy.billaged.v1.BillingService/NotifyPossibleGap"
)

// These variables are the protoreflect.Descriptor objects for the RPCs defined in this package.
var (
	billingServiceServiceDescriptor                 = v1.File_proto_deploy_billaged_v1_billing_proto.Services().ByName("BillingService")
	billingServiceSendMetricsBatchMethodDescriptor  = billingServiceServiceDescriptor.Methods().ByName("SendMetricsBatch")
	billingServiceSendHeartbeatMethodDescriptor     = billingServiceServiceDescriptor.Methods().ByName("SendHeartbeat")
	billingServiceNotifyVmStartedMethodDescriptor   = billingServiceServiceDescriptor.Methods().ByName("NotifyVmStarted")
	billingServiceNotifyVmStoppedMethodDescriptor   = billingServiceServiceDescriptor.Methods().ByName("NotifyVmStopped")
	billingServiceNotifyPossibleGapMethodDescriptor = billingServiceServiceDescriptor.Methods().ByName("NotifyPossibleGap")
)

// BillingServiceClient is a client for the deploy.billaged.v1.BillingService service.
type BillingServiceClient interface {
	SendMetricsBatch(context.Context, *connect.Request[v1.SendMetricsBatchRequest]) (*connect.Response[v1.SendMetricsBatchResponse], error)
	SendHeartbeat(context.Context, *connect.Request[v1.SendHeartbeatRequest]) (*connect.Response[v1.SendHeartbeatResponse], error)
	NotifyVmStarted(context.Context, *connect.Request[v1.NotifyVmStartedRequest]) (*connect.Response[v1.NotifyVmStartedResponse], error)
	NotifyVmStopped(context.Context, *connect.Request[v1.NotifyVmStoppedRequest]) (*connect.Response[v1.NotifyVmStoppedResponse], error)
	NotifyPossibleGap(context.Context, *connect.Request[v1.NotifyPossibleGapRequest]) (*connect.Response[v1.NotifyPossibleGapResponse], error)
}

// NewBillingServiceClient constructs a client for the deploy.billaged.v1.BillingService service. By
// default, it uses the Connect protocol with the binary Protobuf Codec, asks for gzipped responses,
// and sends uncompressed requests. To use the gRPC or gRPC-Web protocols, supply the
// connect.WithGRPC() or connect.WithGRPCWeb() options.
//
// The URL supplied here should be the base URL for the Connect or gRPC server (for example,
// http://api.acme.com or https://acme.com/grpc).
func NewBillingServiceClient(httpClient connect.HTTPClient, baseURL string, opts ...connect.ClientOption) BillingServiceClient {
	baseURL = strings.TrimRight(baseURL, "/")
	return &billingServiceClient{
		sendMetricsBatch: connect.NewClient[v1.SendMetricsBatchRequest, v1.SendMetricsBatchResponse](
			httpClient,
			baseURL+BillingServiceSendMetricsBatchProcedure,
			connect.WithSchema(billingServiceSendMetricsBatchMethodDescriptor),
			connect.WithClientOptions(opts...),
		),
		sendHeartbeat: connect.NewClient[v1.SendHeartbeatRequest, v1.SendHeartbeatResponse](
			httpClient,
			baseURL+BillingServiceSendHeartbeatProcedure,
			connect.WithSchema(billingServiceSendHeartbeatMethodDescriptor),
			connect.WithClientOptions(opts...),
		),
		notifyVmStarted: connect.NewClient[v1.NotifyVmStartedRequest, v1.NotifyVmStartedResponse](
			httpClient,
			baseURL+BillingServiceNotifyVmStartedProcedure,
			connect.WithSchema(billingServiceNotifyVmStartedMethodDescriptor),
			connect.WithClientOptions(opts...),
		),
		notifyVmStopped: connect.NewClient[v1.NotifyVmStoppedRequest, v1.NotifyVmStoppedResponse](
			httpClient,
			baseURL+BillingServiceNotifyVmStoppedProcedure,
			connect.WithSchema(billingServiceNotifyVmStoppedMethodDescriptor),
			connect.WithClientOptions(opts...),
		),
		notifyPossibleGap: connect.NewClient[v1.NotifyPossibleGapRequest, v1.NotifyPossibleGapResponse](
			httpClient,
			baseURL+BillingServiceNotifyPossibleGapProcedure,
			connect.WithSchema(billingServiceNotifyPossibleGapMethodDescriptor),
			connect.WithClientOptions(opts...),
		),
	}
}

// billingServiceClient implements BillingServiceClient.
type billingServiceClient struct {
	sendMetricsBatch  *connect.Client[v1.SendMetricsBatchRequest, v1.SendMetricsBatchResponse]
	sendHeartbeat     *connect.Client[v1.SendHeartbeatRequest, v1.SendHeartbeatResponse]
	notifyVmStarted   *connect.Client[v1.NotifyVmStartedRequest, v1.NotifyVmStartedResponse]
	notifyVmStopped   *connect.Client[v1.NotifyVmStoppedRequest, v1.NotifyVmStoppedResponse]
	notifyPossibleGap *connect.Client[v1.NotifyPossibleGapRequest, v1.NotifyPossibleGapResponse]
}

// SendMetricsBatch calls deploy.billaged.v1.BillingService.SendMetricsBatch.
func (c *billingServiceClient) SendMetricsBatch(ctx context.Context, req *connect.Request[v1.SendMetricsBatchRequest]) (*connect.Response[v1.SendMetricsBatchResponse], error) {
	return c.sendMetricsBatch.CallUnary(ctx, req)
}

// SendHeartbeat calls deploy.billaged.v1.BillingService.SendHeartbeat.
func (c *billingServiceClient) SendHeartbeat(ctx context.Context, req *connect.Request[v1.SendHeartbeatRequest]) (*connect.Response[v1.SendHeartbeatResponse], error) {
	return c.sendHeartbeat.CallUnary(ctx, req)
}

// NotifyVmStarted calls deploy.billaged.v1.BillingService.NotifyVmStarted.
func (c *billingServiceClient) NotifyVmStarted(ctx context.Context, req *connect.Request[v1.NotifyVmStartedRequest]) (*connect.Response[v1.NotifyVmStartedResponse], error) {
	return c.notifyVmStarted.CallUnary(ctx, req)
}

// NotifyVmStopped calls deploy.billaged.v1.BillingService.NotifyVmStopped.
func (c *billingServiceClient) NotifyVmStopped(ctx context.Context, req *connect.Request[v1.NotifyVmStoppedRequest]) (*connect.Response[v1.NotifyVmStoppedResponse], error) {
	return c.notifyVmStopped.CallUnary(ctx, req)
}

// NotifyPossibleGap calls deploy.billaged.v1.BillingService.NotifyPossibleGap.
func (c *billingServiceClient) NotifyPossibleGap(ctx context.Context, req *connect.Request[v1.NotifyPossibleGapRequest]) (*connect.Response[v1.NotifyPossibleGapResponse], error) {
	return c.notifyPossibleGap.CallUnary(ctx, req)
}

// BillingServiceHandler is an implementation of the deploy.billaged.v1.BillingService service.
type BillingServiceHandler interface {
	SendMetricsBatch(context.Context, *connect.Request[v1.SendMetricsBatchRequest]) (*connect.Response[v1.SendMetricsBatchResponse], error)
	SendHeartbeat(context.Context, *connect.Request[v1.SendHeartbeatRequest]) (*connect.Response[v1.SendHeartbeatResponse], error)
	NotifyVmStarted(context.Context, *connect.Request[v1.NotifyVmStartedRequest]) (*connect.Response[v1.NotifyVmStartedResponse], error)
	NotifyVmStopped(context.Context, *connect.Request[v1.NotifyVmStoppedRequest]) (*connect.Response[v1.NotifyVmStoppedResponse], error)
	NotifyPossibleGap(context.Context, *connect.Request[v1.NotifyPossibleGapRequest]) (*connect.Response[v1.NotifyPossibleGapResponse], error)
}

// NewBillingServiceHandler builds an HTTP handler from the service implementation. It returns the
// path on which to mount the handler and the handler itself.
//
// By default, handlers support the Connect, gRPC, and gRPC-Web protocols with the binary Protobuf
// and JSON codecs. They also support gzip compression.
func NewBillingServiceHandler(svc BillingServiceHandler, opts ...connect.HandlerOption) (string, http.Handler) {
	billingServiceSendMetricsBatchHandler := connect.NewUnaryHandler(
		BillingServiceSendMetricsBatchProcedure,
		svc.SendMetricsBatch,
		connect.WithSchema(billingServiceSendMetricsBatchMethodDescriptor),
		connect.WithHandlerOptions(opts...),
	)
	billingServiceSendHeartbeatHandler := connect.NewUnaryHandler(
		BillingServiceSendHeartbeatProcedure,
		svc.SendHeartbeat,
		connect.WithSchema(billingServiceSendHeartbeatMethodDescriptor),
		connect.WithHandlerOptions(opts...),
	)
	billingServiceNotifyVmStartedHandler := connect.NewUnaryHandler(
		BillingServiceNotifyVmStartedProcedure,
		svc.NotifyVmStarted,
		connect.WithSchema(billingServiceNotifyVmStartedMethodDescriptor),
		connect.WithHandlerOptions(opts...),
	)
	billingServiceNotifyVmStoppedHandler := connect.NewUnaryHandler(
		BillingServiceNotifyVmStoppedProcedure,
		svc.NotifyVmStopped,
		connect.WithSchema(billingServiceNotifyVmStoppedMethodDescriptor),
		connect.WithHandlerOptions(opts...),
	)
	billingServiceNotifyPossibleGapHandler := connect.NewUnaryHandler(
		BillingServiceNotifyPossibleGapProcedure,
		svc.NotifyPossibleGap,
		connect.WithSchema(billingServiceNotifyPossibleGapMethodDescriptor),
		connect.WithHandlerOptions(opts...),
	)
	return "/deploy.billaged.v1.BillingService/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case BillingServiceSendMetricsBatchProcedure:
			billingServiceSendMetricsBatchHandler.ServeHTTP(w, r)
		case BillingServiceSendHeartbeatProcedure:
			billingServiceSendHeartbeatHandler.ServeHTTP(w, r)
		case BillingServiceNotifyVmStartedProcedure:
			billingServiceNotifyVmStartedHandler.ServeHTTP(w, r)
		case BillingServiceNotifyVmStoppedProcedure:
			billingServiceNotifyVmStoppedHandler.ServeHTTP(w, r)
		case BillingServiceNotifyPossibleGapProcedure:
			billingServiceNotifyPossibleGapHandler.ServeHTTP(w, r)
		default:
			http.NotFound(w, r)
		}
	})
}

// UnimplementedBillingServiceHandler returns CodeUnimplemented from all methods.
type UnimplementedBillingServiceHandler struct{}

func (UnimplementedBillingServiceHandler) SendMetricsBatch(context.Context, *connect.Request[v1.SendMetricsBatchRequest]) (*connect.Response[v1.SendMetricsBatchResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("deploy.billaged.v1.BillingService.SendMetricsBatch is not implemented"))
}

func (UnimplementedBillingServiceHandler) SendHeartbeat(context.Context, *connect.Request[v1.SendHeartbeatRequest]) (*connect.Response[v1.SendHeartbeatResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("deploy.billaged.v1.BillingService.SendHeartbeat is not implemented"))
}

func (UnimplementedBillingServiceHandler) NotifyVmStarted(context.Context, *connect.Request[v1.NotifyVmStartedRequest]) (*connect.Response[v1.NotifyVmStartedResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("deploy.billaged.v1.BillingService.NotifyVmStarted is not implemented"))
}

func (UnimplementedBillingServiceHandler) NotifyVmStopped(context.Context, *connect.Request[v1.NotifyVmStoppedRequest]) (*connect.Response[v1.NotifyVmStoppedResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("deploy.billaged.v1.BillingService.NotifyVmStopped is not implemented"))
}

func (UnimplementedBillingServiceHandler) NotifyPossibleGap(context.Context, *connect.Request[v1.NotifyPossibleGapRequest]) (*connect.Response[v1.NotifyPossibleGapResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("deploy.billaged.v1.BillingService.NotifyPossibleGap is not implemented"))
}
