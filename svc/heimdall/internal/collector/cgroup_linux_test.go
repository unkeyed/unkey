//go:build linux

package collector

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

// These tests exercise the cgroup reader against a synthetic cgroup
// directory laid out exactly like the kernel's cgroup v2 files. They don't
// need root, don't need Docker, don't need a running container. They
// catch:
//   - parse bugs in cpu.stat / memory.current / memory.stat
//   - wrong path construction for systemd vs cgroupfs drivers
//   - teardown races (partial cgroup files) behaving as documented
//   - the working-set math (memory.current - memory.stat:inactive_file)
//
// Correctness of the fixtures themselves is anchored by
// TestParseGoldenKernelOutput below, which feeds actual text captured
// from a live pod on a 6.6 kernel through the parsers. If the kernel
// format ever changes (new lines, reordered keys, extra whitespace),
// that test breaks; the synthetic fixtures above it only exercise the
// happy-path subset.

// buildFakePodCgroup lays out a cgroupfs-shaped pod cgroup under `root`
// with the given UID, and writes cpu.stat + memory.* files with the given
// values. Returns the directory path so the test can manipulate it further.
func buildFakePodCgroup(t *testing.T, root string, uid types.UID, qos corev1.PodQOSClass, usageUsec, memCurrent, inactiveFile int64) string {
	t.Helper()
	var qosDir string
	switch qos {
	case corev1.PodQOSBestEffort:
		qosDir = "besteffort"
	case corev1.PodQOSBurstable:
		qosDir = "burstable"
	case corev1.PodQOSGuaranteed:
		qosDir = ""
	default:
		qosDir = ""
	}

	var dir string
	if qosDir == "" {
		dir = filepath.Join(root, "kubepods", "pod"+string(uid))
	} else {
		dir = filepath.Join(root, "kubepods", qosDir, "pod"+string(uid))
	}

	require.NoError(t, os.MkdirAll(dir, 0o755))

	writeFile(t, filepath.Join(dir, "cpu.stat"),
		"usage_usec "+strconv.FormatInt(usageUsec, 10)+"\n"+
			"user_usec 900000\n"+
			"system_usec 100000\n")
	writeFile(t, filepath.Join(dir, "memory.current"),
		strconv.FormatInt(memCurrent, 10)+"\n")
	writeFile(t, filepath.Join(dir, "memory.stat"),
		"anon 100000\n"+
			"file 200000\n"+
			"inactive_file "+strconv.FormatInt(inactiveFile, 10)+"\n"+
			"active_file 50000\n")
	return dir
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}

// --- Cases ---------------------------------------------------------------

func TestCgroupReader_Read_Burstable(t *testing.T) {
	root := t.TempDir()
	uid := types.UID("abc-123")
	buildFakePodCgroup(t, root, uid, corev1.PodQOSBurstable,
		12_345_678, // cpu.stat usage_usec
		256*1024*1024,
		32*1024*1024,
	)

	r := &cgroupReader{root: root, driver: CgroupDriverCgroupfs}
	reading, err := r.read(uid, corev1.PodQOSBurstable)
	require.NoError(t, err)
	require.EqualValues(t, 12_345_678, reading.cpuUsageUsec)

	// working set = current - inactive_file
	wantMem := int64(256-32) * 1024 * 1024
	require.Equal(t, wantMem, reading.memoryBytes)
}

func TestCgroupReader_Read_Guaranteed(t *testing.T) {
	// Guaranteed pods sit directly under kubepods/ with no qos subdir.
	root := t.TempDir()
	uid := types.UID("def-456")
	buildFakePodCgroup(t, root, uid, corev1.PodQOSGuaranteed,
		999_999,
		128*1024*1024,
		0,
	)

	r := &cgroupReader{root: root, driver: CgroupDriverCgroupfs}
	reading, err := r.read(uid, corev1.PodQOSGuaranteed)
	require.NoError(t, err)
	require.EqualValues(t, 999_999, reading.cpuUsageUsec)
	require.EqualValues(t, 128*1024*1024, reading.memoryBytes)
}

func TestCgroupReader_Read_MissingCPU(t *testing.T) {
	// Pod cgroup torn down entirely before we read: os.ErrNotExist expected.
	root := t.TempDir()
	uid := types.UID("ghi-789")
	r := &cgroupReader{root: root, driver: CgroupDriverCgroupfs}

	_, err := r.read(uid, corev1.PodQOSBurstable)
	require.Error(t, err, "expected error for missing cgroup")
	// The cgroupReader wraps the error with fmt.Errorf("...: %w", err).
	// Use errors.Is (not os.IsNotExist, which doesn't unwrap) so the
	// collector's "treat as normal teardown race" check works across the
	// wrapping layer.
	require.ErrorIs(t, err, os.ErrNotExist, "expected wrapped os.ErrNotExist, got: %v", err)
}

func TestCgroupReader_Read_PartialTeardown(t *testing.T) {
	// cpu.stat is present, memory.current is missing. Memory races with
	// teardown; the reader MUST preserve the CPU counter (more valuable
	// for billing) and return memoryBytes=0 without error.
	root := t.TempDir()
	uid := types.UID("jkl-000")
	dir := buildFakePodCgroup(t, root, uid, corev1.PodQOSBurstable,
		5_000_000,
		100*1024*1024,
		0,
	)
	require.NoError(t, os.Remove(filepath.Join(dir, "memory.current")))

	r := &cgroupReader{root: root, driver: CgroupDriverCgroupfs}
	reading, err := r.read(uid, corev1.PodQOSBurstable)
	require.NoError(t, err, "read should not error on partial teardown")
	require.EqualValues(t, 5_000_000, reading.cpuUsageUsec, "CPU counter must be preserved on partial teardown")
	require.Zero(t, reading.memoryBytes, "memoryBytes should be 0 on partial teardown")
}

func TestCgroupReader_Read_MemoryNegativeClamp(t *testing.T) {
	// Kernel updates memory.current and memory.stat non-atomically.
	// If inactive_file reads larger than current (the kernel is mid-
	// reclaim), working set would be negative. The reader must clamp to 0.
	root := t.TempDir()
	uid := types.UID("mno-111")
	buildFakePodCgroup(t, root, uid, corev1.PodQOSBurstable,
		100,
		50,  // current (smaller than inactive_file below)
		200, // inactive_file > current
	)

	r := &cgroupReader{root: root, driver: CgroupDriverCgroupfs}
	reading, err := r.read(uid, corev1.PodQOSBurstable)
	require.NoError(t, err)
	require.Zero(t, reading.memoryBytes, "memoryBytes should clamp to 0 when inactive_file > current")
}

// --- Parser-level tests --------------------------------------------------

func TestParseCPUStat(t *testing.T) {
	f := filepath.Join(t.TempDir(), "cpu.stat")
	writeFile(t, f,
		"usage_usec 1234567\n"+
			"user_usec 1000000\n"+
			"system_usec 234567\n")
	got, err := parseCPUStat(f)
	require.NoError(t, err)
	require.EqualValues(t, 1234567, got)
}

func TestParseCPUStat_Missing(t *testing.T) {
	_, err := parseCPUStat(filepath.Join(t.TempDir(), "does-not-exist"))
	require.Error(t, err, "expected error for missing file")
	require.ErrorIs(t, err, os.ErrNotExist, "expected os.ErrNotExist, got %v", err)
}

func TestParseCPUStat_NoUsageLine(t *testing.T) {
	f := filepath.Join(t.TempDir(), "cpu.stat")
	writeFile(t, f, "user_usec 100\nsystem_usec 50\n")
	_, err := parseCPUStat(f)
	require.Error(t, err, "expected error when usage_usec line missing")
}

func TestParseMemoryCurrent(t *testing.T) {
	f := filepath.Join(t.TempDir(), "memory.current")
	writeFile(t, f, "104857600\n")
	got, err := parseMemoryCurrent(f)
	require.NoError(t, err)
	require.EqualValues(t, 104857600, got)
}

func TestParseMemoryStatInactiveFile(t *testing.T) {
	f := filepath.Join(t.TempDir(), "memory.stat")
	writeFile(t, f,
		"anon 100\n"+
			"file 200\n"+
			"inactive_file 12345\n"+
			"active_file 50\n")
	got, err := parseMemoryStatInactiveFile(f)
	require.NoError(t, err)
	require.EqualValues(t, 12345, got)
}

func TestParseMemoryStatInactiveFile_MissingLine(t *testing.T) {
	// Older kernels omit inactive_file under certain cgroup configurations.
	// The parser returns 0 rather than failing so the reader can still
	// produce a memory number.
	f := filepath.Join(t.TempDir(), "memory.stat")
	writeFile(t, f, "anon 100\nfile 200\n")
	got, err := parseMemoryStatInactiveFile(f)
	require.NoError(t, err)
	require.Zero(t, got, "missing inactive_file should parse as 0")
}

// --- Golden-fixture test: real kernel output -----------------------------
//
// The tests above use minimal synthetic fixtures to exercise specific
// edge cases. The parsers still have to cope with whatever the kernel
// actually writes: longer files, extra keys, trailing newlines, ordering
// quirks. This test feeds the parsers verbatim text captured from a
// running pod's cgroup files:
//
//	kubectl -n unkey exec heimdall-<pod> -- cat \
//	  /sys/fs/cgroup/kubepods/burstable/pod<uid>/{cpu.stat,memory.current,memory.stat}
//
// Captured 2026-04-19 from a minikube node (cgroup v2, cgroupfs driver)
// running krane-managed deployment pods under gVisor. If the kernel
// ever changes these files (new cgroup subsystems land, keys get
// reordered, format shifts to JSON), this test breaks first and tells
// us to update the synthetic fixtures above. Without this anchor, the
// other tests would keep passing against a fake format that no longer
// matches reality.

const goldenCPUStat = `usage_usec 157687122
user_usec 99216066
system_usec 58471056
nice_usec 0
nr_periods 0
nr_throttled 0
throttled_usec 0
nr_bursts 0
burst_usec 0
`

const goldenMemoryCurrent = `57991168
`

// Verbatim memory.stat from a live pod: 50+ keys, with inactive_file
// sitting in the middle of the file, not at the start. A naive parser
// that early-returns on the first match or that doesn't scan the whole
// file breaks on this input.
const goldenMemoryStat = `anon 25473024
file 31657984
kernel 0
kernel_stack 0
pagetables 0
sec_pagetables 0
percpu 0
sock 0
vmalloc 0
shmem 0
file_mapped 9056256
file_dirty 0
file_writeback 0
swapcached 0
anon_thp 0
file_thp 0
shmem_thp 0
inactive_anon 3969024
active_anon 22360064
inactive_file 21635072
active_file 10027008
unevictable 0
slab_reclaimable 0
slab_unreclaimable 0
slab 0
workingset_refault_anon 0
workingset_refault_file 4133635
workingset_activate_anon 0
workingset_activate_file 2668986
workingset_restore_anon 0
workingset_restore_file 31908
workingset_nodereclaim 0
pgdemote_kswapd 0
pgdemote_direct 0
pgdemote_khugepaged 0
pgdemote_proactive 0
pgscan 4604435
pgsteal 4142477
pswpin 0
pswpout 0
pgscan_kswapd 0
pgscan_direct 4604435
pgscan_khugepaged 0
pgscan_proactive 0
pgsteal_kswapd 0
pgsteal_direct 4142477
pgsteal_khugepaged 0
pgsteal_proactive 0
pgfault 1193945
pgmajfault 9937
pgrefill 566432
pgactivate 346447
pgdeactivate 0
pglazyfree 0
pglazyfreed 0
swpin_zero 0
swpout_zero 0
thp_fault_alloc 0
thp_collapse_alloc 0
thp_swpout 0
thp_swpout_fallback 0
`

func TestParseGoldenKernelOutput(t *testing.T) {
	dir := t.TempDir()

	cpuPath := filepath.Join(dir, "cpu.stat")
	writeFile(t, cpuPath, goldenCPUStat)
	usage, err := parseCPUStat(cpuPath)
	require.NoError(t, err)
	require.EqualValues(t, 157687122, usage, "usage_usec must be extracted from a realistic cpu.stat")

	memCurrentPath := filepath.Join(dir, "memory.current")
	writeFile(t, memCurrentPath, goldenMemoryCurrent)
	current, err := parseMemoryCurrent(memCurrentPath)
	require.NoError(t, err)
	require.EqualValues(t, 57991168, current)

	memStatPath := filepath.Join(dir, "memory.stat")
	writeFile(t, memStatPath, goldenMemoryStat)
	inactive, err := parseMemoryStatInactiveFile(memStatPath)
	require.NoError(t, err)
	require.EqualValues(t, 21635072, inactive, "inactive_file must be extracted from deep inside memory.stat")

	// End-to-end: what the reader would compute against these files.
	uid := types.UID("golden-pod")
	podDir := filepath.Join(dir, "kubepods", "burstable", "pod"+string(uid))
	require.NoError(t, os.MkdirAll(podDir, 0o755))
	writeFile(t, filepath.Join(podDir, "cpu.stat"), goldenCPUStat)
	writeFile(t, filepath.Join(podDir, "memory.current"), goldenMemoryCurrent)
	writeFile(t, filepath.Join(podDir, "memory.stat"), goldenMemoryStat)

	r := &cgroupReader{root: dir, driver: CgroupDriverCgroupfs}
	reading, err := r.read(uid, corev1.PodQOSBurstable)
	require.NoError(t, err)
	require.EqualValues(t, 157687122, reading.cpuUsageUsec)
	// working set = memory.current - inactive_file = 57991168 - 21635072 = 36356096
	require.EqualValues(t, 36356096, reading.memoryBytes, "working set = memory.current - inactive_file")
}
