package privatenetwork

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"

	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/deploy/privatenetwork"
	"github.com/unkeyed/unkey/svc/krane/pkg/labels"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation"
)

// topologyData validates topology and returns the ConfigMap data that records
// every cell's region.
func (r *Reconciler) topologyData(topology *ctrlv1.PrivateNetworkTopology) (map[string]string, error) {
	if topology.GetVersion() != 1 {
		return nil, fmt.Errorf("unsupported private network topology version %d", topology.GetVersion())
	}
	data := map[string]string{
		"version": "1", "platform": r.clusterKey.GetPlatform(), "cell": r.clusterKey.GetCellId(),
	}
	for _, cluster := range topology.GetClusters() {
		key := "region." + cluster.GetCellId()
		if cluster.GetCellId() == "" || len(validation.IsConfigMapKey(key)) != 0 ||
			cluster.GetPlatform() == "" || cluster.GetPlatform() != r.clusterKey.GetPlatform() ||
			!validRegion(cluster.GetRegion()) {
			return nil, fmt.Errorf("invalid cluster in private network topology")
		}
		if _, exists := data[key]; exists {
			return nil, fmt.Errorf("duplicate cell in private network topology: %s", cluster.GetCellId())
		}
		data[key] = cluster.GetRegion()
	}
	if region := data["region."+r.clusterKey.GetCellId()]; region == "" || region != r.clusterKey.GetRegion() {
		return nil, fmt.Errorf("private network topology does not match this cluster")
	}
	return data, nil
}

// topologyRegions returns the distinct regions in validated topology data.
func topologyRegions(data map[string]string) []string {
	var regions []string
	for key, region := range data {
		if strings.HasPrefix(key, "region.") && !slices.Contains(regions, region) {
			regions = append(regions, region)
		}
	}
	slices.Sort(regions)
	return regions
}

func validRegion(region string) bool {
	return len(validation.IsDNS1123Label(region)) == 0 && len(region) <= privatenetwork.MaxRegionLength && region != "local-first"
}

func (r *Reconciler) ensureTopology(ctx context.Context, topology *ctrlv1.PrivateNetworkTopology) error {
	if topology == nil {
		return nil
	}
	data, err := r.topologyData(topology)
	if err != nil {
		return err
	}

	client := r.client.CoreV1().ConfigMaps(metav1.NamespaceSystem)
	existing, err := client.Get(ctx, privatenetwork.TopologyConfigMap, metav1.GetOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("get private network topology: %w", err)
	}
	l := labels.New().ManagedByKrane().Component(privatenetwork.TopologyComponent)
	desired := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: privatenetwork.TopologyConfigMap, Namespace: metav1.NamespaceSystem, Labels: l},
		Data:       data,
	}
	if apierrors.IsNotFound(err) {
		_, err = client.Create(ctx, desired, metav1.CreateOptions{FieldManager: fieldManager})
	} else {
		if !maps.Equal(existing.Labels, l) {
			return fmt.Errorf("refuse to replace foreign private network topology ConfigMap")
		}
		for key, region := range data {
			if previous, exists := existing.Data[key]; exists && previous != region {
				return fmt.Errorf("private network topology identity changed for %s", key)
			}
		}
		if maps.Equal(existing.Data, data) {
			return nil
		}
		desired.ResourceVersion, desired.UID = existing.ResourceVersion, existing.UID
		_, err = client.Update(ctx, desired, metav1.UpdateOptions{FieldManager: fieldManager})
	}
	if err != nil {
		return fmt.Errorf("publish private network topology: %w", err)
	}
	return nil
}
