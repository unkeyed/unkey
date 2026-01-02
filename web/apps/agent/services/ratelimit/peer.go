package ratelimit

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"connectrpc.com/connect"
	"connectrpc.com/otelconnect"
	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fmsg"
	"github.com/unkeyed/unkey/apps/agent/gen/proto/ratelimit/v1/ratelimitv1connect"
	"github.com/unkeyed/unkey/apps/agent/pkg/cluster"
	"github.com/unkeyed/unkey/apps/agent/pkg/tracing"
)

type authorizedRoundTripper struct {
	rt      http.RoundTripper
	headers http.Header
}

func (h *authorizedRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	for k, vv := range h.headers {
		for _, v := range vv {

			r.Header.Add(k, v)
		}
	}
	return h.rt.RoundTrip(r)
}

func (s *service) getPeerClient(ctx context.Context, key string) (ratelimitv1connect.RatelimitServiceClient, cluster.Node, error) {
	ctx, span := tracing.Start(ctx, "ratelimit.getPeer")
	defer span.End()

	peer, err := s.cluster.FindNode(key)
	if err != nil {
		tracing.RecordError(span, err)
		return nil, peer, fault.Wrap(err, fmsg.With("unable to find responsible nodes"))
	}
	s.consistencyChecker.Record(key, peer.Id)

	url := peer.RpcAddr
	if !strings.Contains(url, "://") {
		url = "http://" + url
	}
	s.peersMu.RLock()
	c, ok := s.peers[url]
	s.peersMu.RUnlock()
	if !ok {
		interceptor, err := otelconnect.NewInterceptor(otelconnect.WithTracerProvider(tracing.GetGlobalTraceProvider()))
		if err != nil {
			tracing.RecordError(span, err)
			s.logger.Err(err).Msg("failed to create interceptor")
			return nil, peer, err
		}
		httpClient := &http.Client{
			Transport: &authorizedRoundTripper{
				rt:      http.DefaultTransport,
				headers: http.Header{"Authorization": []string{fmt.Sprintf("Bearer %s", s.cluster.AuthToken())}},
			},
		}
		c = ratelimitv1connect.NewRatelimitServiceClient(httpClient, url, connect.WithInterceptors(interceptor))
		s.peersMu.Lock()
		s.peers[url] = c
		s.peersMu.Unlock()
	}

	return c, peer, nil
}

type peer struct {
	id     string
	client ratelimitv1connect.RatelimitServiceClient
}

// getAllPeers returns clients for all nodes in the cluster except ourselves
func (s *service) getAllPeers(context.Context) ([]peer, error) {
	peers := []peer{}
	for _, p := range s.cluster.Peers() {
		if p.Id == s.cluster.NodeId() {
			continue
		}
		url := p.RpcAddr
		if !strings.Contains(url, "://") {
			url = "http://" + url
		}
		s.peersMu.RLock()
		c, ok := s.peers[url]
		s.peersMu.RUnlock()
		if !ok {
			interceptor, err := otelconnect.NewInterceptor(otelconnect.WithTracerProvider(tracing.GetGlobalTraceProvider()))
			if err != nil {
				s.logger.Err(err).Msg("failed to create interceptor")
				return nil, err
			}
			httpClient := &http.Client{
				Transport: &authorizedRoundTripper{
					rt:      http.DefaultTransport,
					headers: http.Header{"Authorization": []string{fmt.Sprintf("Bearer %s", s.cluster.AuthToken())}},
				},
			}
			c = ratelimitv1connect.NewRatelimitServiceClient(httpClient, url, connect.WithInterceptors(interceptor))
			s.peersMu.Lock()
			s.peers[url] = c
			s.peersMu.Unlock()
		}
		peers = append(peers, peer{id: p.Id, client: c})
	}
	return peers, nil
}
