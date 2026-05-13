package collector

// CollectorSet enumerates which metric kinds heimdall is scraping. Built
// from the validated Config.Collectors string list at startup so the
// runtime can do bool checks instead of repeated string matches in the
// per-tick hot path.
//
// A disabled collector means: don't read the source (saving the cost),
// and write zero into the corresponding schema columns. The billing
// math (max(counter)-min(counter) over a window) is unaffected — zero
// in, zero out — and InstanceCheckpointAttributes.Collectors records
// the truth so query-time consumers can tell "disabled" from "really
// zero traffic".
type CollectorSet struct {
	CPU     bool
	Memory  bool
	Disk    bool
	Network bool
}

// CollectorSetFrom builds a CollectorSet from a list of collector names.
// Caller (heimdall.Config.Validate) is responsible for ensuring names
// are one of {cpu, memory, disk, network}; unknown names silently no-op
// here so a config typo doesn't crash the agent at runtime.
func CollectorSetFrom(names []string) CollectorSet {
	var cs CollectorSet
	for _, n := range names {
		switch n {
		case "cpu":
			cs.CPU = true
		case "memory":
			cs.Memory = true
		case "disk":
			cs.Disk = true
		case "network":
			cs.Network = true
		}
	}
	return cs
}

// Names returns the canonical list of enabled collector names in a stable
// order. Used to stamp the per-checkpoint attributes column so query-time
// consumers can tell "0 because disabled" from "0 because measured zero".
func (cs CollectorSet) Names() []string {
	out := make([]string, 0, 4)
	if cs.CPU {
		out = append(out, "cpu")
	}
	if cs.Memory {
		out = append(out, "memory")
	}
	if cs.Disk {
		out = append(out, "disk")
	}
	if cs.Network {
		out = append(out, "network")
	}
	return out
}

func (c *Collector) enabledCollectorNames() []string { return c.collectors.Names() }
