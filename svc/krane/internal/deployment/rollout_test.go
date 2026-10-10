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

var gateNow = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

func gateDeployment(name string, ageMinutes int, fixtures ...rolloutFixture) appsv1.Deployment {
	d := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:         "ns",
			Name:              name,
			Generation:        2,
			CreationTimestamp: metav1.NewTime(gateNow.Add(-time.Duration(ageMinutes) * time.Minute)),
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

func stalledAgo(age time.Duration) rolloutFixture {
	return func(d *appsv1.Deployment) {
		d.Annotations = map[string]string{
			rolloutStalledTemplateAnnotation: podTemplateHash(d),
			rolloutStalledAtAnnotation:       gateNow.Add(-age).Format(time.RFC3339),
		}
	}
}

func stalledOnOtherTemplate(d *appsv1.Deployment) {
	d.Annotations = map[string]string{
		rolloutStalledTemplateAnnotation: "other-template",
		rolloutStalledAtAnnotation:       gateNow.Format(time.RFC3339),
	}
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
		stall          []string
		finish         []string
		admit          []string
		admitUnhealthy []string
		slotsHeld      int
		stalled        int
		waiting        int
		halted         bool
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
			name: "stalled rollout is paused and keeps its slot until then",
			deployments: []appsv1.Deployment{
				gateDeployment("stuck", 1, unpaused, withSlot, rollingOut, stalled),
				gateDeployment("next", 1, templateChanged),
			},
			maxConcurrent: 2,
			stall:         []string{"stuck"},
			admit:         []string{"next"},
			slotsHeld:     1,
		},
		{
			name: "recently stalled template does not roll again",
			deployments: []appsv1.Deployment{
				gateDeployment("stuck", 1, templateChanged, stalledAgo(10*time.Minute)),
			},
			maxConcurrent: 5,
			stalled:       1,
		},
		{
			name: "stalled template rolls again after the retry delay",
			deployments: []appsv1.Deployment{
				gateDeployment("stuck", 1, templateChanged, stalledAgo(7*time.Hour)),
			},
			maxConcurrent: 5,
			admit:         []string{"stuck"},
		},
		{
			name: "new template after a stall rolls at once",
			deployments: []appsv1.Deployment{
				gateDeployment("fixed", 1, templateChanged, stalledOnOtherTemplate),
			},
			maxConcurrent: 5,
			admit:         []string{"fixed"},
		},
		{
			name: "halts when the limit of rollouts stalled within the window",
			deployments: []appsv1.Deployment{
				gateDeployment("stuck-a", 1, templateChanged, stalledAgo(10*time.Minute)),
				gateDeployment("stuck-b", 1, unpaused, withSlot, rollingOut, stalled),
				gateDeployment("next", 1, templateChanged),
				gateDeployment("broken", 1, templateChanged, unavailable),
			},
			maxConcurrent: 2,
			stall:         []string{"stuck-b"},
			slotsHeld:     1,
			stalled:       1,
			waiting:       2,
			halted:        true,
		},
		{
			name: "older stalls do not halt the gate",
			deployments: []appsv1.Deployment{
				gateDeployment("stuck", 1, templateChanged, stalledAgo(2*time.Hour)),
				gateDeployment("next", 1, templateChanged),
			},
			maxConcurrent: 1,
			admit:         []string{"next"},
			stalled:       1,
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
			plan := planRollouts(tt.deployments, tt.maxConcurrent, gateNow)

			require.ElementsMatch(t, tt.stall, names(plan.stall), "stall")
			require.ElementsMatch(t, tt.finish, names(plan.finish), "finish")
			require.Equal(t, tt.admit, nilIfEmpty(names(plan.admit)), "admit")
			require.Equal(t, tt.admitUnhealthy, nilIfEmpty(names(plan.admitUnhealthy)), "admitUnhealthy")
			require.Equal(t, tt.slotsHeld, plan.slotsHeld, "slotsHeld")
			require.Equal(t, tt.stalled, plan.stalled, "stalled")
			require.Equal(t, tt.waiting, plan.waiting, "waiting")
			require.Equal(t, tt.halted, plan.halted, "halted")
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

	ctrl.setRolloutState(t.Context(), &d, rolloutState{slot: true})

	got, err := client.AppsV1().Deployments("ns").Get(t.Context(), "dep", metav1.GetOptions{})
	require.NoError(t, err)
	require.False(t, got.Spec.Paused)
	require.True(t, holdsRolloutSlot(got))

	ctrl.setRolloutState(t.Context(), got, rolloutState{paused: true, stalledTemplate: podTemplateHash(got), stalledAt: gateNow})

	got, err = client.AppsV1().Deployments("ns").Get(t.Context(), "dep", metav1.GetOptions{})
	require.NoError(t, err)
	require.True(t, got.Spec.Paused)
	require.NotContains(t, got.Annotations, rolloutSlotAnnotation)
	require.True(t, stalledWithin(got, gateNow.Add(time.Minute), stalledRolloutRetryAfter))

	ctrl.setRolloutState(t.Context(), got, rolloutState{paused: true})

	got, err = client.AppsV1().Deployments("ns").Get(t.Context(), "dep", metav1.GetOptions{})
	require.NoError(t, err)
	require.NotContains(t, got.Annotations, rolloutStalledTemplateAnnotation)
	require.NotContains(t, got.Annotations, rolloutStalledAtAnnotation)
}
