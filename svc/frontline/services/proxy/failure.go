package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"syscall"

	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/hoptracing"
)

type ErrorSource string

const (
	SourceNetwork  ErrorSource = "network"
	SourceSentinel ErrorSource = "sentinel"
	SourceUpstream ErrorSource = "upstream"
)

type ErrorKind string

const (
	KindClientClosed ErrorKind = "client_closed"
	KindTimeout      ErrorKind = "timeout"
	KindConnRefused  ErrorKind = "conn_refused"
	KindConnReset    ErrorKind = "conn_reset"
	KindHostUnreach  ErrorKind = "host_unreachable"
	KindDNSNotFound  ErrorKind = "dns_not_found"
	KindDNSTimeout   ErrorKind = "dns_timeout"
	KindUpstream5xx  ErrorKind = "upstream_5xx"
	KindUnknown      ErrorKind = "unknown"
)

type FailureSpec struct {
	URN           codes.URN
	Status        int
	PublicMessage string
}

type ProxyFailure struct {
	Source         ErrorSource
	Kind           ErrorKind
	URN            codes.URN
	Status         int
	PublicMessage  string
	UpstreamStatus int
	Cause          error
}

var defaultSpecs = map[ErrorKind]FailureSpec{
	KindClientClosed: {
		URN:           codes.User.BadRequest.ClientClosedRequest.URN(),
		Status:        499,
		PublicMessage: "The client closed the connection before the request completed.",
	},
	KindTimeout: {
		URN:           codes.Frontline.Proxy.GatewayTimeout.URN(),
		Status:        http.StatusGatewayTimeout,
		PublicMessage: "The request took too long to process. Please try again later.",
	},
	KindConnRefused: {
		URN:           codes.Frontline.Proxy.ServiceUnavailable.URN(),
		Status:        http.StatusServiceUnavailable,
		PublicMessage: "The service is temporarily unavailable. Please try again later.",
	},
	KindConnReset: {
		URN:           codes.Frontline.Proxy.BadGateway.URN(),
		Status:        http.StatusBadGateway,
		PublicMessage: "Connection was reset by the backend service. Please try again.",
	},
	KindHostUnreach: {
		URN:           codes.Frontline.Proxy.ServiceUnavailable.URN(),
		Status:        http.StatusServiceUnavailable,
		PublicMessage: "The service is unreachable. Please try again later.",
	},
	KindDNSNotFound: {
		URN:           codes.Frontline.Proxy.ServiceUnavailable.URN(),
		Status:        http.StatusServiceUnavailable,
		PublicMessage: "The service could not be found. Please check your configuration.",
	},
	KindDNSTimeout: {
		URN:           codes.Frontline.Proxy.GatewayTimeout.URN(),
		Status:        http.StatusGatewayTimeout,
		PublicMessage: "DNS resolution timed out. Please try again later.",
	},
	KindUpstream5xx: {
		URN:           codes.Frontline.Proxy.BadGateway.URN(),
		Status:        http.StatusBadGateway,
		PublicMessage: "The upstream service returned an error. Please try again.",
	},
	KindUnknown: {
		URN:           codes.Frontline.Proxy.BadGateway.URN(),
		Status:        http.StatusBadGateway,
		PublicMessage: "Unable to connect. Please try again in a few moments.",
	},
}

func fromSpec(src ErrorSource, kind ErrorKind, cause error) ProxyFailure {
	spec := defaultSpecs[kind]
	return ProxyFailure{
		Source:         src,
		Kind:           kind,
		URN:            spec.URN,
		Status:         spec.Status,
		PublicMessage:  spec.PublicMessage,
		UpstreamStatus: 0,
		Cause:          cause,
	}
}

func ClassifyNetworkError(err error) ProxyFailure {
	if errors.Is(err, context.Canceled) {
		return fromSpec(SourceNetwork, KindClientClosed, err)
	}
	if errors.Is(err, context.DeadlineExceeded) || os.IsTimeout(err) {
		return fromSpec(SourceNetwork, KindTimeout, err)
	}

	var netErr *net.OpError
	if errors.As(err, &netErr) {
		if netErr.Timeout() {
			return fromSpec(SourceNetwork, KindTimeout, err)
		}
		switch {
		case errors.Is(netErr.Err, syscall.ECONNREFUSED):
			return fromSpec(SourceNetwork, KindConnRefused, err)
		case errors.Is(netErr.Err, syscall.ECONNRESET):
			return fromSpec(SourceNetwork, KindConnReset, err)
		case errors.Is(netErr.Err, syscall.EHOSTUNREACH):
			return fromSpec(SourceNetwork, KindHostUnreach, err)
		}
	}

	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		if dnsErr.IsNotFound {
			return fromSpec(SourceNetwork, KindDNSNotFound, err)
		}
		if dnsErr.IsTimeout {
			return fromSpec(SourceNetwork, KindDNSTimeout, err)
		}
	}

	return fromSpec(SourceNetwork, KindUnknown, err)
}

type sentinelErrorEnvelope struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func ClassifyUpstreamResponse(resp *http.Response) (*ProxyFailure, bool) {
	if resp.Header.Get(hoptracing.HeaderErrorSource) == "sentinel" {
		body, readErr := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		resp.Body = io.NopCloser(bytes.NewReader(body))

		if readErr == nil {
			var env sentinelErrorEnvelope
			if json.Unmarshal(body, &env) == nil && env.Error.Code != "" {
				msg := env.Error.Message
				if msg == "" {
					msg = "An unexpected error occurred. Please try again later."
				}
				pf := ProxyFailure{
					Source:         SourceSentinel,
					Kind:           KindUnknown,
					URN:            codes.URN(env.Error.Code),
					Status:         resp.StatusCode,
					PublicMessage:  msg,
					UpstreamStatus: resp.StatusCode,
					Cause:          nil,
				}
				return &pf, true
			}
		}

		pf := fromSpec(SourceSentinel, KindUnknown, nil)
		pf.Status = resp.StatusCode
		pf.UpstreamStatus = resp.StatusCode
		return &pf, true
	}

	if resp.StatusCode >= 500 {
		pf := fromSpec(SourceUpstream, KindUpstream5xx, nil)
		pf.UpstreamStatus = resp.StatusCode
		return &pf, true
	}

	return nil, false
}

func (pf ProxyFailure) ToFault(internal string) error {
	return fault.Wrap(pf.Cause,
		fault.Code(pf.URN),
		fault.Internal(internal),
		fault.Public(pf.PublicMessage),
	)
}

func (pf ProxyFailure) IsInfrastructureError() bool {
	switch pf.Source {
	case SourceNetwork:
		return true
	case SourceSentinel:
		return true
	case SourceUpstream:
		return false
	default:
		return false
	}
}
