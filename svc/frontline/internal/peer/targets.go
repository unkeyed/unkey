package peer

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/netip"

	"github.com/unkeyed/unkey/svc/krane/pkg/labels"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

var errUnknownTarget = errors.New("peer: target is not a live local instance of the deployment")

// TargetValidator decides whether the relay may dial address for deploymentID.
type TargetValidator interface {
	Validate(deploymentID, address string) error
}

func targetAddr(address string) (netip.Addr, error) {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return netip.Addr{}, err
	}
	addr, err := netip.ParseAddr(host)
	if err != nil {
		return netip.Addr{}, err
	}
	return addr.Unmap(), nil
}

// CIDRTargets allows any address inside the prefixes. It does not check the
// deployment, so it is only for environments without a pod index.
type CIDRTargets []netip.Prefix

// ParseCIDRTargets parses CIDR strings.
func ParseCIDRTargets(cidrs []string) (CIDRTargets, error) {
	out := make(CIDRTargets, 0, len(cidrs))
	for _, raw := range cidrs {
		prefix, err := netip.ParsePrefix(raw)
		if err != nil {
			return nil, fmt.Errorf("parse target cidr %q: %w", raw, err)
		}
		out = append(out, prefix.Masked())
	}
	return out, nil
}

func (c CIDRTargets) Validate(_, address string) error {
	addr, err := targetAddr(address)
	if err != nil {
		return err
	}
	for _, prefix := range c {
		if prefix.Contains(addr) {
			return nil
		}
	}
	return errUnknownTarget
}

const podIPIndex = "podIP"

// PodTargets validates targets against the local cluster's running krane
// pods: the address must be the IP of a running, non-terminating pod labeled
// with the deployment ID. This stops stale addresses that now belong to
// another tenant's pod and any dial outside customer pods.
type PodTargets struct {
	indexer cache.Indexer
}

// NewPodTargets starts a pod informer and waits for its first sync.
func NewPodTargets(ctx context.Context, client kubernetes.Interface) (*PodTargets, error) {
	selector := fmt.Sprintf("%s=krane,%s=deployment", labels.LabelKeyManagedBy, labels.LabelKeyComponent)
	factory := informers.NewSharedInformerFactoryWithOptions(client, 0,
		informers.WithTweakListOptions(func(o *metav1.ListOptions) { o.LabelSelector = selector }),
	)
	informer := factory.Core().V1().Pods().Informer()
	if err := informer.AddIndexers(cache.Indexers{podIPIndex: indexPodIPs}); err != nil {
		return nil, fmt.Errorf("add pod ip index: %w", err)
	}
	factory.Start(ctx.Done())
	if !cache.WaitForCacheSync(ctx.Done(), informer.HasSynced) {
		return nil, fmt.Errorf("pod informer did not sync")
	}
	return &PodTargets{indexer: informer.GetIndexer()}, nil
}

func indexPodIPs(obj any) ([]string, error) {
	pod, ok := obj.(*corev1.Pod)
	if !ok {
		return nil, nil
	}
	ips := make([]string, 0, len(pod.Status.PodIPs))
	for _, ip := range pod.Status.PodIPs {
		if addr, err := netip.ParseAddr(ip.IP); err == nil {
			ips = append(ips, addr.Unmap().String())
		}
	}
	return ips, nil
}

func (p *PodTargets) Validate(deploymentID, address string) error {
	addr, err := targetAddr(address)
	if err != nil {
		return err
	}
	objs, err := p.indexer.ByIndex(podIPIndex, addr.String())
	if err != nil {
		return err
	}
	for _, obj := range objs {
		pod, ok := obj.(*corev1.Pod)
		if !ok {
			continue
		}
		if pod.DeletionTimestamp == nil &&
			pod.Status.Phase == corev1.PodRunning &&
			pod.Labels[labels.LabelKeyDeploymentID] == deploymentID {
			return nil
		}
	}
	return errUnknownTarget
}
