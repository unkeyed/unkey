package main

import (
	"context"
	"fmt"
	"slices"

	"github.com/unkeyed/unkey/pkg/deploy/deploygate"
	dbtype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/pkg/uid"
)

func checkDeploygate(ctx context.Context) (result, error) {
	r := result{Name: "deploygate", Cases: 0, Violations: 0, Counterexample: nil}
	statuses := append(slices.Clone(dbtype.AllDeploymentStatuses), "", "unknown")
	desiredStates := []dbtype.DeploymentsDesiredState{"running", "stopped", "", "unknown"}
	environments := []dbtype.EnvironmentKind{"production", "preview", "", "unknown"}
	target := uid.New(uid.DeploymentPrefix)
	currentIDs := []string{"", target, uid.New(uid.DeploymentPrefix)}
	for _, status := range statuses {
		for _, desired := range desiredStates {
			for _, environment := range environments {
				if err := ctx.Err(); err != nil {
					return r, err
				}
				state := fmt.Sprintf("status=%q desired=%q environment=%q", status, desired, environment)
				stopAllowed := status == "ready" && desired == "running" && environment != "production"
				err := deploygate.CheckStopTarget(deploygate.StopInput{
					Status: status, DesiredState: desired, EnvironmentKind: environment,
				})
				r.observe([]string{state, "stop"}, admissionMismatch(stopAllowed, err))
				for relation, current := range currentIDs {
					eligible := status == "ready" && desired == "running" && environment == "production" && relation != 0
					trace := []string{state, fmt.Sprintf("current relation=%d (0=absent, 1=target, 2=other)", relation)}
					err = deploygate.CheckRollbackTarget(deploygate.RollbackInput{
						Status: status, DesiredState: desired, EnvironmentKind: environment,
						CurrentDeploymentID: current, DeploymentID: target,
					})
					r.observe(append(trace, "rollback"), admissionMismatch(eligible && relation != 1, err))
					for _, rolledBack := range []bool{false, true} {
						err = deploygate.CheckPromoteTarget(deploygate.PromoteInput{
							Status: status, DesiredState: desired, EnvironmentKind: environment,
							CurrentDeploymentID: current, DeploymentID: target, IsRolledBack: rolledBack,
						})
						r.observe(append(trace, fmt.Sprintf("promote rolledBack=%t", rolledBack)), admissionMismatch(eligible && (relation != 1 || rolledBack), err))
					}
				}
			}
		}
	}
	for _, desired := range desiredStates {
		for _, environment := range environments {
			for _, suspended := range []bool{false, true} {
				err := deploygate.CheckStartTarget(deploygate.StartInput{
					DesiredState: desired, EnvironmentKind: environment, SpendSuspended: suspended,
				})
				allowed := desired == "stopped" && environment != "production" && !suspended
				r.observe([]string{fmt.Sprintf("start desired=%q environment=%q suspended=%t", desired, environment, suspended)}, admissionMismatch(allowed, err))
			}
		}
	}
	return r, nil
}

func admissionMismatch(allowed bool, err error) string {
	if allowed == (err == nil) {
		return ""
	}
	return fmt.Sprintf("admitted=%t, want %t (error=%v)", err == nil, allowed, err)
}
