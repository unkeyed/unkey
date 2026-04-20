//go:build linux

package network

import (
	"encoding/binary"
	"net"
	"testing"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/rlimit"
	"github.com/stretchr/testify/require"
)

// These tests load the compiled BPF program and invoke it synthetically
// via BPF_PROG_TEST_RUN (cilium/ebpf's Program.Run). Each case constructs
// a fake L2 + L3 frame with a chosen source/destination IP, runs one of
// count_egress / count_ingress on it, and asserts the right slot in the
// counter map moved by skb->len.
//
// The test intentionally doesn't exercise the Go-side attach / containerd
// / netlink paths: those depend on a real pod and a real kernel interface.
// This test is the classifier's unit-level contract only.
//
// Requires CAP_BPF (or root) + a Linux kernel >= 5.8 to load cgroup_skb-era
// BPF programs. On CI runners without that, the tests self-skip.

// --- Frame builders -------------------------------------------------------
//
// Byte offsets below are derived from the relevant RFCs and sliced from a
// local buffer so each access reads as the RFC offset, not a sum. That is:
//   eth  = frame[0:14]       (dst MAC | src MAC | ethertype)
//   ip4  = frame[14:34]      (IPv4 header, 20 bytes minimum)
//   ip6  = frame[14:54]      (IPv6 header, fixed 40 bytes)
// Every offset into those sub-slices lines up with RFC 791 / RFC 8200.

const (
	ethHeaderLen         = 14
	ipv4HeaderLen        = 20
	ipv6HeaderLen        = 40
	ethTypeIPv4   uint16 = 0x0800
	ethTypeIPv6   uint16 = 0x86DD
	protoUDP      byte   = 17
)

// writeEthernetHeader writes a minimal Ethernet frame header: all-zero
// MAC addresses, the given ethertype at bytes 12-13. Real MACs don't
// matter to the classifier.
func writeEthernetHeader(frame []byte, ethType uint16) {
	binary.BigEndian.PutUint16(frame[12:14], ethType)
}

// buildIPv4Frame returns an Ethernet+IPv4 frame with the given
// source/destination addresses. Total length is ethHeaderLen+ipv4HeaderLen
// at minimum; excess is zero-padded to satisfy skb->len for the classifier.
func buildIPv4Frame(t *testing.T, src, dst net.IP, totalLen int) []byte {
	t.Helper()
	src4 := src.To4()
	dst4 := dst.To4()
	require.NotNil(t, src4, "src must be a v4 address: %v", src)
	require.NotNil(t, dst4, "dst must be a v4 address: %v", dst)

	if totalLen < ethHeaderLen+ipv4HeaderLen {
		totalLen = ethHeaderLen + ipv4HeaderLen
	}
	frame := make([]byte, totalLen)

	writeEthernetHeader(frame, ethTypeIPv4)

	// IPv4 header (RFC 791). Offsets below are relative to the header
	// start, which is slice-anchored at frame[ethHeaderLen:].
	ip := frame[ethHeaderLen : ethHeaderLen+ipv4HeaderLen]
	ip[0] = 0x45                                                       // version=4, IHL=5 (20-byte header)
	binary.BigEndian.PutUint16(ip[2:4], uint16(totalLen-ethHeaderLen)) // total_length
	ip[8] = 64                                                         // TTL
	ip[9] = protoUDP                                                   // any non-zero protocol; the classifier doesn't read L4
	copy(ip[12:16], src4)
	copy(ip[16:20], dst4)
	return frame
}

// buildIPv6Frame returns an Ethernet+IPv6 frame with the given
// source/destination addresses.
func buildIPv6Frame(t *testing.T, src, dst net.IP, totalLen int) []byte {
	t.Helper()
	src16 := src.To16()
	dst16 := dst.To16()
	require.NotNil(t, src16, "src must be a v6 address: %v", src)
	require.NotNil(t, dst16, "dst must be a v6 address: %v", dst)

	if totalLen < ethHeaderLen+ipv6HeaderLen {
		totalLen = ethHeaderLen + ipv6HeaderLen
	}
	frame := make([]byte, totalLen)

	writeEthernetHeader(frame, ethTypeIPv6)

	// IPv6 header (RFC 8200). 40 bytes fixed.
	ip := frame[ethHeaderLen : ethHeaderLen+ipv6HeaderLen]
	ip[0] = 0x60 // version=6 (high nibble), traffic class low nibble=0
	// bytes 4-5 = payload length (header is 40 of the 54+ byte frame)
	// The classifier ignores it so zero is fine.
	ip[6] = protoUDP // next header
	ip[7] = 64       // hop limit
	copy(ip[8:24], src16)
	copy(ip[24:40], dst16)
	return frame
}

// --- Helpers --------------------------------------------------------------

// loadTestObjects loads a fresh, process-local copy of the BPF program +
// counter map. Each test gets its own instance so results don't bleed.
func loadTestObjects(t *testing.T) *bpfObjects {
	t.Helper()
	if err := rlimit.RemoveMemlock(); err != nil {
		t.Skipf("rlimit.RemoveMemlock unavailable (likely no CAP_SYS_RESOURCE): %v", err)
	}
	objs := &bpfObjects{} //nolint:exhaustruct // populated by loadBpfObjects
	// The BPF program declares `pod_counters` with LIBBPF_PIN_BY_NAME,
	// which requires a PinPath when loading. Give each test a tempdir so
	// runs don't share state with production or each other.
	opts := &ebpf.CollectionOptions{ //nolint:exhaustruct
		Maps: ebpf.MapOptions{PinPath: t.TempDir()}, //nolint:exhaustruct
	}
	if err := loadBpfObjects(objs, opts); err != nil {
		t.Skipf("cannot load BPF objects (kernel or privileges missing): %v", err)
	}
	t.Cleanup(func() { _ = objs.Close() })
	return objs
}

// runProg invokes a BPF program on a single frame via BPF_PROG_TEST_RUN.
// The test assumes TC_ACT_UNSPEC (-1) as the return code since our program
// is an observer and must be non-terminating in the TCX chain.
func runProg(t *testing.T, prog *ebpf.Program, frame []byte) {
	t.Helper()
	ret, err := prog.Run(&ebpf.RunOptions{Data: frame}) //nolint:exhaustruct
	require.NoError(t, err, "prog.Run")
	require.Equal(t, int32(-1), int32(ret), "observer must return TC_ACT_UNSPEC")
}

// readCounters fetches the map entry for the synthetic ifindex=0 that
// BPF_PROG_TEST_RUN uses by default.
func readCounters(t *testing.T, m *ebpf.Map) bpfCounters {
	t.Helper()
	var c bpfCounters
	err := m.Lookup(uint32(0), &c)
	require.NoError(t, err, "map lookup ifindex=0")
	return c
}

// --- Cases ----------------------------------------------------------------

func TestClassifier_V4_PublicEgress(t *testing.T) {
	objs := loadTestObjects(t)
	const frameLen = 1000
	frame := buildIPv4Frame(t,
		net.IPv4(10, 244, 0, 5),  // pod
		net.IPv4(5, 161, 7, 195), // Hetzner, public
		frameLen,
	)

	runProg(t, objs.CountEgress, frame)

	c := readCounters(t, objs.PodCounters)
	require.EqualValues(t, frameLen, c.EgressPublic, "egress_public should match skb->len")
	require.Zero(t, c.EgressPrivate, "egress_private")
	require.Zero(t, c.IngressPublic, "ingress_public")
	require.Zero(t, c.IngressPrivate, "ingress_private")
}

func TestClassifier_V4_PrivateEgress(t *testing.T) {
	objs := loadTestObjects(t)
	const frameLen = 500
	frame := buildIPv4Frame(t,
		net.IPv4(10, 244, 0, 5),
		net.IPv4(10, 96, 0, 10), // coredns ClusterIP, RFC1918 10.0.0.0/8
		frameLen,
	)

	runProg(t, objs.CountEgress, frame)

	c := readCounters(t, objs.PodCounters)
	require.EqualValues(t, frameLen, c.EgressPrivate, "egress_private")
	require.Zero(t, c.EgressPublic, "egress_public should be 0 for RFC1918")
}

func TestClassifier_V4_PublicIngress(t *testing.T) {
	objs := loadTestObjects(t)
	const frameLen = 1500
	frame := buildIPv4Frame(t,
		net.IPv4(5, 161, 7, 195), // public source
		net.IPv4(10, 244, 0, 5),  // pod destination
		frameLen,
	)

	runProg(t, objs.CountIngress, frame)

	c := readCounters(t, objs.PodCounters)
	require.EqualValues(t, frameLen, c.IngressPublic, "ingress classifier looks at saddr")
}

func TestClassifier_V4_CGNAT(t *testing.T) {
	// 100.64.0.0/10 is CGNAT and must classify as private.
	objs := loadTestObjects(t)
	frame := buildIPv4Frame(t,
		net.IPv4(10, 244, 0, 5),
		net.IPv4(100, 64, 0, 1),
		500,
	)

	runProg(t, objs.CountEgress, frame)

	c := readCounters(t, objs.PodCounters)
	require.EqualValues(t, 500, c.EgressPrivate, "100.64.0.0/10 is CGNAT (private)")
}

func TestClassifier_V4_NotCGNAT(t *testing.T) {
	// 100.128.0.0 is one bit past CGNAT's /10 boundary: public.
	objs := loadTestObjects(t)
	frame := buildIPv4Frame(t,
		net.IPv4(10, 244, 0, 5),
		net.IPv4(100, 128, 0, 1),
		500,
	)

	runProg(t, objs.CountEgress, frame)

	c := readCounters(t, objs.PodCounters)
	require.EqualValues(t, 500, c.EgressPublic, "100.128.0.0 is outside CGNAT /10")
}

func TestClassifier_V4_Loopback(t *testing.T) {
	// 127.0.0.0/8 is loopback, private.
	objs := loadTestObjects(t)
	frame := buildIPv4Frame(t,
		net.IPv4(10, 244, 0, 5),
		net.IPv4(127, 0, 0, 1),
		500,
	)

	runProg(t, objs.CountEgress, frame)

	c := readCounters(t, objs.PodCounters)
	require.EqualValues(t, 500, c.EgressPrivate, "127.0.0.0/8 is loopback (private)")
}

func TestClassifier_V6_ULA(t *testing.T) {
	objs := loadTestObjects(t)
	frame := buildIPv6Frame(t,
		net.ParseIP("fd00::1"),
		net.ParseIP("fd00::2"), // fc00::/7 ULA, private
		800,
	)

	runProg(t, objs.CountEgress, frame)

	c := readCounters(t, objs.PodCounters)
	require.EqualValues(t, 800, c.EgressPrivate, "fc00::/7 ULA is private")
}

func TestClassifier_V6_PublicEgress(t *testing.T) {
	objs := loadTestObjects(t)
	frame := buildIPv6Frame(t,
		net.ParseIP("fd00::1"),
		net.ParseIP("2001:4860:4860::8888"), // Google DNS, public
		800,
	)

	runProg(t, objs.CountEgress, frame)

	c := readCounters(t, objs.PodCounters)
	require.EqualValues(t, 800, c.EgressPublic, "2001:4860::/32 is public")
}

func TestClassifier_V6_LinkLocal(t *testing.T) {
	objs := loadTestObjects(t)
	frame := buildIPv6Frame(t,
		net.ParseIP("fe80::1"),
		net.ParseIP("fe80::2"), // fe80::/10 link-local
		600,
	)

	runProg(t, objs.CountEgress, frame)

	c := readCounters(t, objs.PodCounters)
	require.EqualValues(t, 600, c.EgressPrivate, "fe80::/10 is link-local (private)")
}

func TestClassifier_V6_Multicast(t *testing.T) {
	objs := loadTestObjects(t)
	frame := buildIPv6Frame(t,
		net.ParseIP("fe80::1"),
		net.ParseIP("ff02::1"), // ff00::/8 multicast, not billable egress
		300,
	)

	runProg(t, objs.CountEgress, frame)

	c := readCounters(t, objs.PodCounters)
	require.EqualValues(t, 300, c.EgressPrivate, "ff00::/8 multicast is private")
}

func TestClassifier_V6_Loopback(t *testing.T) {
	objs := loadTestObjects(t)
	frame := buildIPv6Frame(t,
		net.ParseIP("::1"),
		net.ParseIP("::1"),
		200,
	)

	runProg(t, objs.CountEgress, frame)

	c := readCounters(t, objs.PodCounters)
	require.EqualValues(t, 200, c.EgressPrivate, "::1/128 loopback is private")
}

func TestClassifier_NonIP(t *testing.T) {
	// ARP (ethertype 0x0806) is not IP. The classifier must do nothing
	// (return TC_ACT_UNSPEC, no map entry created).
	objs := loadTestObjects(t)
	frame := make([]byte, 64)
	writeEthernetHeader(frame, 0x0806)

	runProg(t, objs.CountEgress, frame)

	var c bpfCounters
	err := objs.PodCounters.Lookup(uint32(0), &c)
	// Either the map lookup errors with ErrKeyNotExist (no entry created)
	// OR the entry exists with all zeros. Both are acceptable.
	if err == nil {
		require.Zero(t, c.EgressPublic)
		require.Zero(t, c.EgressPrivate)
	}
}

func TestClassifier_Accumulates(t *testing.T) {
	// Three identical packets should sum exactly 3 * frameLen into the
	// egress_public slot. This is the idempotency-of-sum property the
	// billing pipeline relies on for counter monotonicity.
	objs := loadTestObjects(t)
	const frameLen = 1000
	frame := buildIPv4Frame(t,
		net.IPv4(10, 244, 0, 5),
		net.IPv4(5, 161, 7, 195),
		frameLen,
	)

	for i := 0; i < 3; i++ {
		runProg(t, objs.CountEgress, frame)
	}

	c := readCounters(t, objs.PodCounters)
	require.EqualValues(t, 3*frameLen, c.EgressPublic, "counters must accumulate")
}
