// Package dbdiag is the shared MySQL diagnostics layer for preflight
// probe Diagnose() implementations. It owns the small handful of
// "look up the deployment row + its step history" queries that
// almost every failure investigation needs, so the probes themselves
// do not duplicate raw SQL or sqlc plumbing.
//
// All public methods return ([]core.Artifact, nil) on best-effort
// success and ([]core.Artifact{<error_note>}, nil) on lookup failure.
// They never return an error to the caller, because Diagnose() is
// itself best-effort and a failed lookup should not mask the real
// probe failure that triggered diagnosis in the first place.
package dbdiag

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/svc/preflight/core"
)

// Diagnoser captures deployment-related rows for inclusion in a
// failure artifact bundle. Construct one per probe call (cheap; just
// holds a *db.Database reference) or share one — there is no state.
type Diagnoser struct {
	db db.Database
}

// New returns a Diagnoser backed by handle's read-only replica.
// Returns nil when handle is nil so callers can do
// `dbdiag.New(env.DB).CaptureBySHA(...)` without a separate nil check.
func New(handle db.Database) *Diagnoser {
	if handle == nil {
		return nil
	}
	return &Diagnoser{db: handle}
}

// CaptureBySHA looks up the most recent deployment row for the given
// commit SHA, then captures it plus its step history as artifacts.
// The looked-up deployment ID is returned separately so callers can
// chain into k8sdiag without a second DB roundtrip. Empty SHA returns
// (nil, "").
func (d *Diagnoser) CaptureBySHA(ctx context.Context, sha string) (artifacts []core.Artifact, deploymentID string) {
	if d == nil || sha == "" {
		return nil, ""
	}

	dep, err := db.Query.FindLatestDeploymentByCommitSha(ctx, d.db.RO(), sql.NullString{String: sha, Valid: true})
	if err != nil {
		return []core.Artifact{noteArtifact("deployment_lookup.txt",
			fmt.Sprintf("FindLatestDeploymentByCommitSha(%q): %v\n", sha, err))}, ""
	}

	return d.captureDeployment(ctx, dep), dep.ID
}

// CaptureByID looks up a deployment by its ID. Useful for probes that
// already know the deployment ID (CreateDeployment, rollback, etc).
func (d *Diagnoser) CaptureByID(ctx context.Context, id string) []core.Artifact {
	if d == nil || id == "" {
		return nil
	}

	dep, err := db.Query.FindDeploymentById(ctx, d.db.RO(), id)
	if err != nil {
		return []core.Artifact{noteArtifact("deployment_lookup.txt",
			fmt.Sprintf("FindDeploymentById(%q): %v\n", id, err))}
	}
	return d.captureDeployment(ctx, dep)
}

// captureDeployment serialises the deployment row plus its full step
// history. Splitting into two artifacts keeps the bundle's grep
// surface clear: deployment_row.json for high-level state, steps.json
// for the timeline.
func (d *Diagnoser) captureDeployment(ctx context.Context, dep db.Deployment) []core.Artifact {
	artifacts := make([]core.Artifact, 0, 2)

	if depJSON, err := json.MarshalIndent(dep, "", "  "); err == nil {
		artifacts = append(artifacts, core.Artifact{
			Name:        "deployment_row.json",
			ContentType: "application/json",
			Body:        depJSON,
		})
	}

	steps, err := db.Query.ListDeploymentStepsByDeploymentId(ctx, d.db.RO(), dep.ID)
	if err != nil {
		artifacts = append(artifacts, noteArtifact("deployment_steps.txt",
			fmt.Sprintf("ListDeploymentStepsByDeploymentId(%q): %v\n", dep.ID, err)))
		return artifacts
	}
	if len(steps) > 0 {
		if stepsJSON, err := json.MarshalIndent(steps, "", "  "); err == nil {
			artifacts = append(artifacts, core.Artifact{
				Name:        "deployment_steps.json",
				ContentType: "application/json",
				Body:        stepsJSON,
			})
		}
	}

	return artifacts
}

func noteArtifact(name, body string) core.Artifact {
	return core.Artifact{
		Name:        name,
		ContentType: "text/plain",
		Body:        []byte(body),
	}
}
