package cluster

import (
	"testing"

	"github.com/unkeyed/unkey/pkg/db"
)

func TestSentinelConverged(t *testing.T) {
	// baseline: all convergence conditions satisfied.
	base := db.FindSentinelDeployContextByK8sNameRow{
		ID:              "snt_1",
		DeployStatus:    db.SentinelsDeployStatusProgressing,
		DesiredImage:    "registry.io/sentinel:v2",
		RunningImage:    "registry.io/sentinel:v2",
		DesiredReplicas: 3,
	}

	tests := []struct {
		name              string
		mutate            func(r *db.FindSentinelDeployContextByK8sNameRow)
		health            db.SentinelsHealth
		availableReplicas int32
		want              bool
	}{
		{
			name:              "all dimensions match",
			mutate:            func(_ *db.FindSentinelDeployContextByK8sNameRow) {},
			health:            db.SentinelsHealthHealthy,
			availableReplicas: 3,
			want:              true,
		},
		{
			name:              "available exceeds desired is ok",
			mutate:            func(_ *db.FindSentinelDeployContextByK8sNameRow) {},
			health:            db.SentinelsHealthHealthy,
			availableReplicas: 5,
			want:              true,
		},
		{
			name: "not progressing",
			mutate: func(r *db.FindSentinelDeployContextByK8sNameRow) {
				r.DeployStatus = db.SentinelsDeployStatusReady
			},
			health:            db.SentinelsHealthHealthy,
			availableReplicas: 3,
			want:              false,
		},
		{
			name:              "unhealthy",
			mutate:            func(_ *db.FindSentinelDeployContextByK8sNameRow) {},
			health:            db.SentinelsHealthUnhealthy,
			availableReplicas: 3,
			want:              false,
		},
		{
			name: "running image empty",
			mutate: func(r *db.FindSentinelDeployContextByK8sNameRow) {
				r.RunningImage = ""
			},
			health:            db.SentinelsHealthHealthy,
			availableReplicas: 3,
			want:              false,
		},
		{
			name: "image mismatch",
			mutate: func(r *db.FindSentinelDeployContextByK8sNameRow) {
				r.RunningImage = "registry.io/sentinel:v1"
			},
			health:            db.SentinelsHealthHealthy,
			availableReplicas: 3,
			want:              false,
		},
		{
			// Regression guard: image matches but replicas are still scaling
			// up. Old predicate would fire convergence prematurely.
			name:              "replica scale-up not yet complete",
			mutate:            func(_ *db.FindSentinelDeployContextByK8sNameRow) {},
			health:            db.SentinelsHealthHealthy,
			availableReplicas: 2,
			want:              false,
		},
		{
			name: "desired replicas zero with zero available",
			mutate: func(r *db.FindSentinelDeployContextByK8sNameRow) {
				r.DesiredReplicas = 0
			},
			health:            db.SentinelsHealthHealthy,
			availableReplicas: 0,
			want:              true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			row := base
			tt.mutate(&row)
			got := sentinelConverged(row, tt.health, tt.availableReplicas)
			if got != tt.want {
				t.Fatalf("sentinelConverged(%+v, %s, %d) = %v, want %v",
					row, tt.health, tt.availableReplicas, got, tt.want)
			}
		})
	}
}
