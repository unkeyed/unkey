package undns

import (
	"context"
	"net"

	dnswire "codeberg.org/miekg/dns"
	"codeberg.org/miekg/dns/dnsutil"
)

func (h *handler) acquireForward(workspace string) bool {
	select {
	case h.forwards <- struct{}{}:
	default:
		h.shed.WithLabelValues("forwards").Inc()
		return false
	}

	h.workspaceMu.Lock()
	defer h.workspaceMu.Unlock()
	limit, metric := h.config.ForwardsPerWorkspace, "forwards_per_workspace"
	if workspace == "" {
		limit, metric = h.config.ForwardsUnidentified, "forwards_unidentified"
	}
	if h.workspaceForwards[workspace] >= limit {
		<-h.forwards
		h.shed.WithLabelValues(metric).Inc()
		return false
	}
	h.workspaceForwards[workspace]++
	return true
}

func (h *handler) releaseForward(workspace string) {
	h.workspaceMu.Lock()
	if h.workspaceForwards[workspace] <= 1 {
		delete(h.workspaceForwards, workspace)
	} else {
		h.workspaceForwards[workspace]--
	}
	h.workspaceMu.Unlock()
	<-h.forwards
}

func (h *handler) forward(ctx context.Context, request *dnswire.Msg, transport, workspace string) *dnswire.Msg {
	if !h.acquireForward(workspace) {
		response := new(dnswire.Msg)
		dnsutil.SetReply(response, request)
		response.Rcode = dnswire.RcodeServerFailure
		response.RecursionAvailable = true
		return response
	}
	defer h.releaseForward(workspace)

	query := dnswire.NewMsg(request.Question[0].Header().Name, dnswire.RRToType(request.Question[0]))
	query.RecursionDesired = request.RecursionDesired
	query.UDPSize = 1232

	dialer := new(net.Dialer)
	dialer.Timeout = h.config.ForwardTimeout
	transportConfig := dnswire.NewTransport()
	transportConfig.Dialer = dialer
	transportConfig.ReadTimeout = h.config.ForwardTimeout
	transportConfig.WriteTimeout = h.config.ForwardTimeout
	client := dnswire.NewClient()
	client.Transport = transportConfig

	ctx, cancel := context.WithTimeout(ctx, h.config.ForwardTimeout)
	defer cancel()

	response, _, err := client.Exchange(ctx, query, transport, h.config.Upstream)
	if err == nil && response.Truncated && transport == "udp" {
		response, _, err = client.Exchange(ctx, query, "tcp", h.config.Upstream)
	}

	if err != nil || response == nil || !response.Response || response.Opcode != dnswire.OpcodeQuery ||
		len(response.Question) != 1 || !sameQuestion(response.Question[0], query.Question[0]) {
		response = new(dnswire.Msg)
		dnsutil.SetReply(response, request)
		response.Rcode = dnswire.RcodeServerFailure
		response.RecursionAvailable = true
		return response
	}

	response.ID = request.ID
	return response
}

func sameQuestion(a, b dnswire.RR) bool {
	return a.Header().Name == b.Header().Name && a.Header().Class == b.Header().Class && dnswire.RRToType(a) == dnswire.RRToType(b)
}
