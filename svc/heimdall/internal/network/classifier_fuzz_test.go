//go:build linux

package network

import (
	"errors"
	"syscall"
	"testing"

	"github.com/cilium/ebpf"
)

// FuzzClassifier feeds arbitrary byte slices through the loaded BPF programs
// via BPF_PROG_TEST_RUN. The properties we want to hold for every input:
//
//  1. Either the kernel rejects the run (small or otherwise invalid skb), or
//     the program returns. We never see a deadlock or kernel panic. The
//     verifier rejects unbounded loops and out-of-bounds reads at load time,
//     so any kernel-side rejection here is a frame-size or context-shape
//     issue, not a classifier bug.
//  2. When the program returns successfully, the return value is always
//     TC_ACT_UNSPEC (-1). We're an observer in the TCX chain; any other
//     return value would change packet handling.
//
// What we are NOT trying to verify:
//
//   - Counter math against the input. Reimplementing the classifier in Go
//     just to compare counters back would just port the same bug if it
//     existed; the verifier already proves we don't read OOB and the unit
//     tests cover the parsing branches deterministically.
//
// Seed corpus mirrors the deterministic test cases (public/private v4 and
// v6, plus a few minimal frames) so the fuzzer starts with inputs known to
// reach every branch.
func FuzzClassifier(f *testing.F) {
	// Reuse the deterministic test frames as seeds. These exercise every
	// classify branch (public/private × v4/v6 × egress/ingress).
	f.Add(buildSeedFrame(t4Public()))
	f.Add(buildSeedFrame(t4Private()))
	f.Add(buildSeedFrame(t6Public()))
	f.Add(buildSeedFrame(t6Private()))
	// Minimal frame sizes that the kernel sometimes accepts; let the
	// fuzzer explore the size dimension from very small upward.
	f.Add([]byte{})
	f.Add(make([]byte, 14)) // ethernet header only, no IP
	f.Add(make([]byte, 34)) // ethernet + minimum v4 header

	f.Fuzz(func(t *testing.T, frame []byte) {
		objs := loadFuzzObjects(t)
		for _, prog := range []*ebpf.Program{objs.CountEgress, objs.CountIngress} {
			ret, err := prog.Run(&ebpf.RunOptions{Data: frame}) //nolint:exhaustruct
			if err != nil {
				// EINVAL covers the "skb too small" / "context shape rejected"
				// cases the kernel emits before our program runs. Those are
				// kernel-side rejections of the test invocation, not bugs in
				// the classifier itself.
				if errors.Is(err, syscall.EINVAL) {
					continue
				}
				t.Fatalf("prog.Run unexpected error: %v (frame_len=%d)", err, len(frame))
			}
			if int32(ret) != -1 {
				t.Fatalf("classifier must return TC_ACT_UNSPEC (-1); got %d for frame_len=%d", int32(ret), len(frame))
			}
		}
	})
}

// loadFuzzObjects mirrors loadTestObjects but skips on env errors silently
// (CI runners without CAP_BPF should skip the fuzz harness, not fail).
func loadFuzzObjects(t *testing.T) *bpfObjects {
	t.Helper()
	objs := &bpfObjects{}            //nolint:exhaustruct
	opts := &ebpf.CollectionOptions{ //nolint:exhaustruct
		Maps: ebpf.MapOptions{PinPath: bpffsTempDir(t)}, //nolint:exhaustruct
	}
	if err := loadBpfObjects(objs, opts); err != nil {
		if isEnvSkipError(err) {
			t.Skipf("cannot load BPF objects (kernel or privileges missing): %v", err)
		}
		t.Fatalf("loadBpfObjects failed: %v", err)
	}
	t.Cleanup(func() { _ = objs.Close() })
	return objs
}

// Seed-frame helpers. Hard-coded byte slices instead of buildIPv4Frame so we
// don't carry the t.Helper() requirement into f.Add (which only takes scalar
// args, not a *testing.T).
type seedFrame struct {
	v6  bool
	src [16]byte
	dst [16]byte
	pad int // total target frame length
}

func t4Public() seedFrame {
	return seedFrame{
		src: ipv4ToSeed(10, 244, 0, 5),
		dst: ipv4ToSeed(5, 161, 7, 195), // public Hetzner IP
		pad: 1000,
	}
}

func t4Private() seedFrame {
	return seedFrame{
		src: ipv4ToSeed(10, 244, 0, 5),
		dst: ipv4ToSeed(10, 96, 0, 10), // RFC1918
		pad: 500,
	}
}

func t6Public() seedFrame {
	return seedFrame{
		v6:  true,
		src: ipv6ToSeed(0xfd, 0x00), // ULA pod side
		dst: ipv6ToSeed(0x2a, 0x06), // public 2a06::/16
		pad: 1500,
	}
}

func t6Private() seedFrame {
	return seedFrame{
		v6:  true,
		src: ipv6ToSeed(0xfd, 0x00),
		dst: ipv6ToSeed(0xfd, 0x01), // ULA fd00::/8
		pad: 800,
	}
}

func ipv4ToSeed(a, b, c, d byte) [16]byte {
	var out [16]byte
	out[0], out[1], out[2], out[3] = a, b, c, d
	return out
}

func ipv6ToSeed(a, b byte) [16]byte {
	var out [16]byte
	out[0], out[1] = a, b
	return out
}

func buildSeedFrame(s seedFrame) []byte {
	if s.v6 {
		if s.pad < ethHeaderLen+ipv6HeaderLen {
			s.pad = ethHeaderLen + ipv6HeaderLen
		}
		frame := make([]byte, s.pad)
		// EtherType = IPv6 (0x86DD) at bytes 12-13.
		frame[12], frame[13] = 0x86, 0xDD
		ip := frame[ethHeaderLen : ethHeaderLen+ipv6HeaderLen]
		ip[0] = 0x60
		ip[6] = protoUDP
		ip[7] = 64
		copy(ip[8:24], s.src[:])
		copy(ip[24:40], s.dst[:])
		return frame
	}
	if s.pad < ethHeaderLen+ipv4HeaderLen {
		s.pad = ethHeaderLen + ipv4HeaderLen
	}
	frame := make([]byte, s.pad)
	frame[12], frame[13] = 0x08, 0x00 // EtherType = IPv4
	ip := frame[ethHeaderLen : ethHeaderLen+ipv4HeaderLen]
	ip[0] = 0x45
	ip[8] = 64
	ip[9] = protoUDP
	copy(ip[12:16], s.src[:4])
	copy(ip[16:20], s.dst[:4])
	return frame
}
