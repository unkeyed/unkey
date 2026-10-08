package deployment

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

type rolloutFixture func(d *appsv1.Deployment)

func gateDeployment(name string, ageMinutes int, fixtures ...rolloutFixture) appsv1.Deployment {
	d := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:         "ns",
			Name:              name,
			Generation:        2,
			CreationTimestamp: metav1.NewTime(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC).Add(-time.Duration(ageMinutes) * time.Minute)),
		},
		Spec: appsv1.DeploymentSpec{Replicas: new(int32(2)), Paused: true},
		Status: appsv1.DeploymentStatus{
			ObservedGeneration: 2,
			Replicas:           2,
			UpdatedReplicas:    2,
			AvailableReplicas:  2,
		},
	}
	for _, fixture := range fixtures {
		fixture(&d)
	}
	return d
}

func unpaused(d *appsv1.Deployment) { d.Spec.Paused = false }

func withSlot(d *appsv1.Deployment) {
	d.Annotations = map[string]string{rolloutSlotAnnotation: "true"}
}

func templateChanged(d *appsv1.Deployment) { d.Status.UpdatedReplicas = 0 }

func rollingOut(d *appsv1.Deployment) {
	d.Status.Replicas = 3
	d.Status.UpdatedReplicas = 1
}

func unavailable(d *appsv1.Deployment) { d.Status.AvailableReplicas = 0 }

func staleStatus(d *appsv1.Deployment) { d.Status.ObservedGeneration = 1 }

func stalled(d *appsv1.Deployment) {
	d.Status.Conditions = []appsv1.DeploymentCondition{{
		Type:   appsv1.DeploymentProgressing,
		Status: corev1.ConditionFalse,
		Reason: progressDeadlineExceededReason,
	}}
}

func names(deployments []*appsv1.Deployment) []string {
	out := make([]string, 0, len(deployments))
	for _, d := range deployments {
		out = append(out, d.Name)
	}
	return out
}

func TestPlanRollouts(t *testing.T) {
	for _, tt := range []struct {
		name           string
		deployments    []appsv1.Deployment
		maxConcurrent  int
		finish         []string
		admit          []string
		admitUnhealthy []string
		slotsHeld      int
		stalled        int
		waiting        int
	}{
		{
			name:          "idle deployments need nothing",
			deployments:   []appsv1.Deployment{gateDeployment("idle", 1)},
			maxConcurrent: 2,
		},
		{
			name: "admits oldest waiting deployments up to the free slots",
			deployments: []appsv1.Deployment{
				gateDeployment("rolling", 1, unpaused, withSlot, rollingOut),
				gateDeployment("young", 1, templateChanged),
				gateDeployment("old", 9, templateChanged),
				gateDeployment("middle", 5, templateChanged),
			},
			maxConcurrent: 3,
			admit:         []string{"old", "middle"},
			slotsHeld:     1,
			waiting:       1,
		},
		{
			name: "finished rollout keeps its slot until it is paused",
			deployments: []appsv1.Deployment{
				gateDeployment("done", 1, unpaused, withSlot),
				gateDeployment("next", 1, templateChanged),
			},
			maxConcurrent: 1,
			finish:        []string{"done"},
			slotsHeld:     1,
			waiting:       1,
		},
		{
			name: "stalled rollout holds its slot",
			deployments: []appsv1.Deployment{
				gateDeployment("stuck", 1, unpaused, withSlot, rollingOut, stalled),
				gateDeployment("next", 1, templateChanged),
			},
			maxConcurrent: 1,
			slotsHeld:     1,
			stalled:       1,
			waiting:       1,
		},
		{
			name: "unhealthy waiting deployments roll without a slot",
			deployments: []appsv1.Deployment{
				gateDeployment("rolling", 1, unpaused, withSlot, rollingOut),
				gateDeployment("broken-a", 2, templateChanged, unavailable),
				gateDeployment("broken-b", 1, templateChanged, unavailable),
			},
			maxConcurrent:  1,
			admitUnhealthy: []string{"broken-a"},
			slotsHeld:      1,
			waiting:        1,
		},
		{
			name: "new workload without a slot does not count, and pauses when done",
			deployments: []appsv1.Deployment{
				gateDeployment("starting", 1, unpaused, rollingOut),
				gateDeployment("started", 1, unpaused),
				gateDeployment("next", 1, templateChanged),
			},
			maxConcurrent: 1,
			finish:        []string{"started"},
			admit:         []string{"next"},
		},
		{
			name: "stale status is not trusted",
			deployments: []appsv1.Deployment{
				gateDeployment("unobserved", 1, templateChanged, staleStatus),
				gateDeployment("unobserved-done", 1, unpaused, withSlot, staleStatus),
			},
			maxConcurrent: 5,
			slotsHeld:     1,
		},
		{
			name:          "zero limit admits nothing healthy",
			deployments:   []appsv1.Deployment{gateDeployment("next", 1, templateChanged)},
			maxConcurrent: 0,
			waiting:       1,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			plan := planRollouts(tt.deployments, tt.maxConcurrent)

			require.ElementsMatch(t, tt.finish, names(plan.finish), "finish")
			require.Equal(t, tt.admit, nilIfEmpty(names(plan.admit)), "admit")
			require.Equal(t, tt.admitUnhealthy, nilIfEmpty(names(plan.admitUnhealthy)), "admitUnhealthy")
			require.Equal(t, tt.slotsHeld, plan.slotsHeld, "slotsHeld")
			require.Equal(t, tt.stalled, plan.stalled, "stalled")
			require.Equal(t, tt.waiting, plan.waiting, "waiting")
		})
	}
}

func nilIfEmpty(s []string) []string {
	if len(s) == 0 {
		return nil
	}
	return s
}

func TestSetRolloutState(t *testing.T) {
	d := gateDeployment("dep", 1, templateChanged)
	client := fake.NewClientset(&d)
	ctrl := &Controller{clientSet: client}

	ctrl.setRolloutState(t.Context(), &d, false, true)

	got, err := client.AppsV1().Deployments("ns").Get(t.Context(), "dep", metav1.GetOptions{})
	require.NoError(t, err)
	require.False(t, got.Spec.Paused)
	require.True(t, holdsRolloutSlot(got))

	ctrl.setRolloutState(t.Context(), got, true, false)

	got, err = client.AppsV1().Deployments("ns").Get(t.Context(), "dep", metav1.GetOptions{})
	require.NoError(t, err)
	require.True(t, got.Spec.Paused)
	require.NotContains(t, got.Annotations, rolloutSlotAnnotation)
}
