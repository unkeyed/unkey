//go:build !linux

package collector

// CgroupDriver is a placeholder matching the Linux build's type. Off-Linux
// builds never use it (preflight returns Unknown + no error).
type CgroupDriver int

const (
	CgroupDriverUnknown CgroupDriver = iota
	CgroupDriverSystemd
	CgroupDriverCgroupfs
)

func (d CgroupDriver) String() string { return "unknown" }

// Preflight is a no-op on non-Linux platforms.
func Preflight(_ string) (CgroupDriver, error) {
	return CgroupDriverUnknown, nil
}
