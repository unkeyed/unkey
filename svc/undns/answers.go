package undns

import (
	"math/rand/v2"
	"net"
	"net/netip"
	"strings"

	dnswire "codeberg.org/miekg/dns"
	"codeberg.org/miekg/dns/dnsutil"
	"codeberg.org/miekg/dns/rdata"
)

func (h *handler) answer(request *dnswire.Msg, remote net.Addr) (*dnswire.Msg, outcome) {
	response := new(dnswire.Msg)
	dnsutil.SetReply(response, request)
	response.RecursionAvailable = true

	if len(request.Question) != 1 || request.Response {
		response.Rcode = dnswire.RcodeFormatError
		return response, outcome{reason: reasonMalformed, err: nil, caller: emptyCaller()}
	}

	question := request.Question[0]
	if request.Opcode != dnswire.OpcodeQuery || question.Header().Class != dnswire.ClassINET {
		response.Rcode = dnswire.RcodeNotImplemented
		return response, outcome{reason: reasonUnsupportedOpcode, err: nil, caller: emptyCaller()}
	}

	podsHealthy := h.catalog.pods.healthy()
	identity, identifyErr := emptyCaller(), errUnknownCaller
	if podsHealthy {
		if ip, err := netip.ParseAddrPort(remote.String()); err == nil {
			identity, identifyErr = h.catalog.identify(ip.Addr())
		}
	}

	name := strings.ToLower(question.Header().Name)
	if !dnsutil.IsBelow(privateZone, name) {
		return nil, outcome{reason: reasonUpstream, err: nil, caller: identity}
	}

	if !podsHealthy {
		response.Rcode = dnswire.RcodeServerFailure
		return response, outcome{reason: reasonIdentityUnavailable, err: nil, caller: emptyCaller()}
	}

	if identifyErr != nil {
		response.Rcode = dnswire.RcodeRefused
		return response, outcome{reason: failureReason(identifyErr), err: identifyErr, caller: emptyCaller()}
	}

	if !h.catalog.ready() {
		response.Rcode = dnswire.RcodeServerFailure
		return response, outcome{reason: reasonDiscoveryNotReady, err: nil, caller: identity}
	}

	return response, h.answerPrivate(response, question, identity)
}

func (h *handler) answerPrivate(response *dnswire.Msg, question dnswire.RR, identity caller) outcome {
	name := strings.ToLower(question.Header().Name)
	qtype := dnswire.RRToType(question)
	response.Authoritative = true
	app := strings.TrimSuffix(name, "."+privateZone)
	result := outcome{reason: reasonAnswer, err: nil, caller: identity}

	if name == privateZone {
		if qtype == dnswire.TypeSOA {
			response.Answer = []dnswire.RR{h.soa()}
		} else {
			response.Ns = []dnswire.RR{h.soa()}
			result.reason = reasonNoData
		}
		return result
	}

	if strings.Contains(app, ".") {
		response.Rcode = dnswire.RcodeNameError
		response.Ns = []dnswire.RR{h.soa()}
		result.reason = reasonUnknownName
		return result
	}

	addresses, exists, err := h.catalog.resolve(identity, app)
	if err != nil {
		response.Rcode = dnswire.RcodeServerFailure
		result.reason, result.err = failureReason(err), err
		return result
	}

	if !exists {
		response.Rcode = dnswire.RcodeNameError
		response.Ns = []dnswire.RR{h.soa()}
		result.reason = reasonUnknownName
		return result
	}

	if qtype != dnswire.TypeA {
		if qtype == dnswire.TypeAAAA {
			response.Ns = []dnswire.RR{h.soa()}
			result.reason = reasonNoData
		} else {
			response.Rcode = dnswire.RcodeNotImplemented
			result.reason = reasonUnsupportedType
		}
		return result
	}

	rand.Shuffle(len(addresses), func(i, j int) { addresses[i], addresses[j] = addresses[j], addresses[i] })
	for _, address := range addresses {
		response.Answer = append(response.Answer, &dnswire.A{
			Hdr: dnswire.Header{Name: question.Header().Name, Class: dnswire.ClassINET, TTL: h.config.TTLSeconds},
			A:   rdata.A{Addr: address},
		})
	}
	return result
}

func (h *handler) soa() *dnswire.SOA {
	return &dnswire.SOA{
		Hdr: dnswire.Header{Name: privateZone, Class: dnswire.ClassINET, TTL: h.config.TTLSeconds},
		SOA: rdata.SOA{
			Ns: "ns." + privateZone, Mbox: "hostmaster." + privateZone, Serial: 1,
			Refresh: 30, Retry: 5, Expire: 60, Minttl: h.config.TTLSeconds,
		},
	}
}
