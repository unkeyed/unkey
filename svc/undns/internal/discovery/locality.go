package discovery

import (
	"github.com/unkeyed/unkey/pkg/deploy/privatenetwork"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation"
)

func (c *Catalog) clusterRegions() (map[string]string, error) {
	object, exists, err := c.topology.GetStore().GetByKey(metav1.NamespaceSystem + "/" + privatenetwork.TopologyConfigMap)
	if err != nil || !exists {
		return nil, err
	}

	config := object.(*corev1.ConfigMap)
	data := config.Data
	if config.DeletionTimestamp != nil || config.Labels[privatenetwork.ManagedByLabel] != "krane" ||
		config.Labels[privatenetwork.ComponentLabel] != privatenetwork.TopologyComponent ||
		data["version"] != "1" || data["platform"] == "" || data["cell"] == "" ||
		len(validation.IsDNS1123Label(data["region."+data["cell"]])) != 0 {
		return nil, nil
	}
	return data, nil
}

func sliceMatchesRegion(slice *discoveryv1.EndpointSlice, topology map[string]string, selector string) bool {
	cell := slice.Labels[privatenetwork.SourceClusterLabel]
	if cell == "" {
		if slice.Labels[discoveryv1.LabelManagedBy] != privatenetwork.SourceSliceManager {
			return false
		}
		if selector == "local-first" {
			return true
		}
		cell = topology["cell"]
	}

	region := topology["region."+cell]
	if len(validation.IsDNS1123Label(region)) != 0 || region == "local-first" {
		return false
	}
	if selector == "local-first" {
		return region == topology["region."+topology["cell"]]
	}
	return region == selector
}
