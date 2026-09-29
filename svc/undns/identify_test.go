package undns

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
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
			other := callerPod("10.0.0.99", "preview")
			other.Name, other.UID = "other", "other"
			tc.mutate(other)
			require.NoError(t, c.pods.GetStore().Add(other))
			require.NoError(t, c.pods.GetStore().Add(callerPod("10.0.0.99", "production")))

			identity, err := c.identify(netipAddress("10.0.0.99"))
			require.Equal(t, tc.wantOK, err == nil, "identify(10.0.0.99) with a running caller and a %s: %v", tc.name, err)
			if tc.wantOK {
				require.Equal(t, "production", identity.kind)
			}
		})
	}
}

func TestIdentifyReindexesPodThatBecomesTerminal(t *testing.T) {
	c := catalogForTest()
	old := callerPod("10.0.0.99", "preview")
	old.Name, old.UID = "old", "old"
	require.NoError(t, c.pods.GetStore().Add(old))
	identity, err := c.identify(netipAddress("10.0.0.99"))
	require.NoError(t, err)
	require.Equal(t, "preview", identity.kind)

	evicted := old.DeepCopy()
	evicted.Status.Phase, evicted.Status.Reason = corev1.PodFailed, "Evicted"
	require.NoError(t, c.pods.GetStore().Update(evicted))
	_, err = c.identify(netipAddress("10.0.0.99"))
	require.ErrorIs(t, err, errUnknownCaller, "identify(10.0.0.99) accepted an evicted pod")

	require.NoError(t, c.pods.GetStore().Add(callerPod("10.0.0.99", "production")))
	identity, err = c.identify(netipAddress("10.0.0.99"))
	require.NoError(t, err, "identify(10.0.0.99) rejected the pod that reused an evicted pod's IP")
	require.Equal(t, "production", identity.kind)
}

func TestIdentifyRequiresDeploymentID(t *testing.T) {
	c := catalogForTest()
	pod := callerPod("10.0.0.99", "production")
	delete(pod.Labels, labels.LabelKeyDeploymentID)
	require.NoError(t, c.pods.GetStore().Add(pod))
	_, err := c.identify(netipAddress("10.0.0.99"))
	require.ErrorIs(t, err, errIneligibleCaller)
}
