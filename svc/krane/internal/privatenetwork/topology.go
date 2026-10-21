package privatenetwork

import (
	"fmt"
	"maps"
	"slices"

	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/deploy/privatenetwork"
	"k8s.io/apimachinery/pkg/util/validation"
)

func (r *Reconciler) topologyRegions(topology *ctrlv1.PrivateNetworkTopology) ([]string, error) {
	if topology == nil {
		return nil, nil
	}
	if topology.GetVersion() != 1 {
		return nil, fmt.Errorf("unsupported private network topology version %d", topology.GetVersion())
	}
	cells := make(map[string]string)
	regions := make(map[string]struct{})
	for _, cluster := range topology.GetClusters() {
		if cluster.GetCellId() == "" ||
			cluster.GetPlatform() == "" || cluster.GetPlatform() != r.clusterKey.GetPlatform() ||
			!validRegion(cluster.GetRegion()) {
			return nil, fmt.Errorf("invalid cluster in private network topology")
		}
		if _, exists := cells[cluster.GetCellId()]; exists {
			return nil, fmt.Errorf("duplicate cell in private network topology: %s", cluster.GetCellId())
		}
		cells[cluster.GetCellId()] = cluster.GetRegion()
		regions[cluster.GetRegion()] = struct{}{}
	}
	if region := cells[r.clusterKey.GetCellId()]; region == "" || region != r.clusterKey.GetRegion() {
		return nil, fmt.Errorf("private network topology does not match this cluster")
	}
	return slices.Sorted(maps.Keys(regions)), nil
}

func validRegion(region string) bool {
	return len(validation.IsDNS1123Label(region)) == 0 && len(region) <= privatenetwork.MaxRegionLength && region != "local-first"
}
