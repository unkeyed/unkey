package undns

import (
	"github.com/unkeyed/unkey/pkg/deploy/appconnection"
	"github.com/unkeyed/unkey/svc/krane/pkg/labels"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation"
)

func (c *catalog) clusterRegions() (map[string]string, error) {
	object, exists, err := c.topology.GetStore().GetByKey(metav1.NamespaceSystem + "/" + appconnection.TopologyConfigMap)
	if err != nil || !exists {
		return nil, err
	}
	config := object.(*corev1.ConfigMap)
	data := config.Data
	if config.DeletionTimestamp != nil || config.Labels[labels.LabelKeyManagedBy] != "krane" ||
		config.Labels[labels.LabelKeyComponent] != appconnection.TopologyComponent ||
		data["version"] != "1" || data["platform"] == "" || data["cell"] == "" ||
		len(validation.IsDNS1123Label(data["region."+data["cell"]])) != 0 {
		return nil, nil
	}
	return data, nil
}

func sliceMatchesRegion(slice *discoveryv1.EndpointSlice, topology map[string]string, selector string) bool {
	cell := slice.Labels["multicluster.kubernetes.io/source-cluster"]
	if cell == "" {
		if slice.Labels[discoveryv1.LabelManagedBy] != "private-dns.unkey.com" {
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
