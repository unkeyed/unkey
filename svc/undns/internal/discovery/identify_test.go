package discovery

import (
	"net/netip"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/krane/pkg/labels"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestIdentifyWithReusedPodIP(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*corev1.Pod)
		wantOK bool
	}{
		{"failed pod with podIP", func(p *corev1.Pod) { p.Status.Phase, p.Status.PodIPs = corev1.PodFailed, nil }, true},
		{"failed pod with podIPs", func(p *corev1.Pod) { p.Status.Phase, p.Status.PodIP = corev1.PodFailed, "" }, true},
		{"succeeded pod with podIP", func(p *corev1.Pod) { p.Status.Phase, p.Status.PodIPs = corev1.PodSucceeded, nil }, true},
		{"succeeded pod with podIPs", func(p *corev1.Pod) { p.Status.Phase, p.Status.PodIP = corev1.PodSucceeded, "" }, true},
		{"pending pod", func(p *corev1.Pod) { p.Status.Phase = corev1.PodPending }, false},
		{"unknown pod", func(p *corev1.Pod) { p.Status.Phase = corev1.PodUnknown }, false},
		{"running pod", func(*corev1.Pod) {}, false},
		{"terminating running pod", func(p *corev1.Pod) { p.DeletionTimestamp = &metav1.Time{Time: time.Now()} }, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := catalogForTest()
			caller := testCaller()
			otherCaller := caller
			otherCaller.Deployment = uid.New(uid.DeploymentPrefix)
			other := callerPod("10.0.0.99", "preview", otherCaller)
			other.Name = "other"
			tc.mutate(other)
			require.NoError(t, c.pods.GetStore().Add(other))
			require.NoError(t, c.pods.GetStore().Add(callerPod("10.0.0.99", "production", caller)))

			identity, err := c.Identify(netipAddress("10.0.0.99"))
			require.Equal(t, tc.wantOK, err == nil, "identify(10.0.0.99) with a running caller and a %s: %v", tc.name, err)
			if tc.wantOK {
				require.Equal(t, caller.Deployment, identity.Deployment)
			}
		})
	}
}

func TestIdentifyReindexesPodThatBecomesTerminal(t *testing.T) {
	c := catalogForTest()
	oldCaller := testCaller()
	old := callerPod("10.0.0.99", "preview", oldCaller)
	old.Name = "old"
	require.NoError(t, c.pods.GetStore().Add(old))
	identity, err := c.Identify(netipAddress("10.0.0.99"))
	require.NoError(t, err)
	require.Equal(t, oldCaller.Deployment, identity.Deployment)

	evicted := old.DeepCopy()
	evicted.Status.Phase, evicted.Status.Reason = corev1.PodFailed, "Evicted"
	require.NoError(t, c.pods.GetStore().Update(evicted))
	_, err = c.Identify(netipAddress("10.0.0.99"))
	requireReason(t, err, ReasonUnknownCaller, "identify(10.0.0.99) accepted an evicted pod")

	newCaller := oldCaller
	newCaller.Deployment = uid.New(uid.DeploymentPrefix)
	require.NoError(t, c.pods.GetStore().Add(callerPod("10.0.0.99", "production", newCaller)))
	identity, err = c.Identify(netipAddress("10.0.0.99"))
	require.NoError(t, err, "identify(10.0.0.99) rejected the pod that reused an evicted pod's IP")
	require.Equal(t, newCaller, identity)
}

func TestIdentifyRequiresDeploymentID(t *testing.T) {
	c := catalogForTest()
	pod := callerPod("10.0.0.99", "production", testCaller())
	delete(pod.Labels, labels.LabelKeyDeploymentID)
	require.NoError(t, c.pods.GetStore().Add(pod))
	_, err := c.Identify(netipAddress("10.0.0.99"))
	requireReason(t, err, ReasonIneligibleCaller)
}

func TestIdentifyRejectsUnknownAndAcceptsPreview(t *testing.T) {
	c := catalogForTest()
	_, err := c.Identify(netipAddress("10.0.0.99"))
	requireReason(t, err, ReasonUnknownCaller)

	pod := callerPod("10.0.0.99", "preview", testCaller())
	require.NoError(t, c.pods.GetStore().Add(pod))
	_, err = c.Identify(netipAddress("10.0.0.99"))
	require.NoError(t, err)

	pod.DeletionTimestamp = &metav1.Time{Time: time.Now()}
	require.NoError(t, c.pods.GetStore().Update(pod))
	_, err = c.Identify(netipAddress("10.0.0.99"))
	requireReason(t, err, ReasonIneligibleCaller)
}

// TestIdentifyRequiresKnownEnvironmentKind guarantees that a Pod outside a
// production or preview environment never receives a caller identity.
func TestIdentifyRequiresKnownEnvironmentKind(t *testing.T) {
	for _, kind := range []string{"", "staging"} {
		c := catalogForTest()
		require.NoError(t, c.pods.GetStore().Add(callerPod("10.0.0.99", kind, testCaller())))
		_, err := c.Identify(netipAddress("10.0.0.99"))
		requireReason(t, err, ReasonIneligibleCaller, "environment kind %q", kind)
	}
}

func netipAddress(raw string) netip.Addr {
	return netip.MustParseAddr(raw)
}
