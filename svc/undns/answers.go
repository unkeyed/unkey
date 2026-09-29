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

func (h *handler) answer(request *dnswire.Msg, remote net.Addr) (*dnswire.Msg, string) {
	response := new(dnswire.Msg)
	dnsutil.SetReply(response, request)
	response.RecursionAvailable = true

	if len(request.Question) != 1 || request.Response {
		response.Rcode = dnswire.RcodeFormatError
		return response, ""
	}

	question := request.Question[0]
	if request.Opcode != dnswire.OpcodeQuery || question.Header().Class != dnswire.ClassINET {
		response.Rcode = dnswire.RcodeNotImplemented
		return response, ""
	}

	podsHealthy := h.catalog.pods.healthy()
	identity, identified := emptyCaller(), false
	if podsHealthy {
		if ip, err := netip.ParseAddrPort(remote.String()); err == nil {
			identity, identified = h.catalog.identify(ip.Addr())
		}
	}

	name := strings.ToLower(question.Header().Name)
	if !dnsutil.IsBelow(privateZone, name) {
		return nil, identity.workspace
	}

	if !podsHealthy {
		response.Rcode = dnswire.RcodeServerFailure
		return response, ""
	}

	if !identified {
		response.Rcode = dnswire.RcodeRefused
		return response, ""
	}

	if !h.catalog.ready() {
		response.Rcode = dnswire.RcodeServerFailure
		return response, ""
	}

	h.answerPrivate(response, question, identity)
	return response, ""
}

func (h *handler) answerPrivate(response *dnswire.Msg, question dnswire.RR, identity caller) {
	name := strings.ToLower(question.Header().Name)
	qtype := dnswire.RRToType(question)
	response.Authoritative = true
	app := strings.TrimSuffix(name, "."+privateZone)

	if name == privateZone {
		if qtype == dnswire.TypeSOA {
			response.Answer = []dnswire.RR{h.soa()}
		} else {
			response.Ns = []dnswire.RR{h.soa()}
		}
		return
	}

	if strings.Contains(app, ".") {
		response.Rcode = dnswire.RcodeNameError
		response.Ns = []dnswire.RR{h.soa()}
		return
	}

	addresses, exists, err := h.catalog.resolve(identity, app)
	if err != nil {
		response.Rcode = dnswire.RcodeServerFailure
		return
	}

	if !exists {
		response.Rcode = dnswire.RcodeNameError
		response.Ns = []dnswire.RR{h.soa()}
		return
	}

	if qtype != dnswire.TypeA {
		if qtype == dnswire.TypeAAAA {
			response.Ns = []dnswire.RR{h.soa()}
		} else {
			response.Rcode = dnswire.RcodeNotImplemented
		}
		return
	}

	rand.Shuffle(len(addresses), func(i, j int) { addresses[i], addresses[j] = addresses[j], addresses[i] })
	for _, address := range addresses {
		response.Answer = append(response.Answer, &dnswire.A{
			Hdr: dnswire.Header{Name: question.Header().Name, Class: dnswire.ClassINET, TTL: h.config.TTLSeconds},
			A:   rdata.A{Addr: address},
		})
	}
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
