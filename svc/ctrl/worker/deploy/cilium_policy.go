package deploy

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	ciliumv2 "github.com/cilium/cilium/pkg/k8s/apis/cilium.io/v2"
	slim_metav1 "github.com/cilium/cilium/pkg/k8s/slim/k8s/apis/meta/v1"
	"github.com/cilium/cilium/pkg/policy/api"
	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/krane/pkg/labels"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ciliumPolicySpec holds everything needed to upsert a single Cilium policy row.
type ciliumPolicySpec struct {
	k8sName      string
	k8sNamespace string
	policy       ciliumv2.CiliumNetworkPolicy
}

// ensureCiliumNetworkPolicy persists or updates Cilium policies for all deployment regions.
//
// Two policies are created per deployment per region:
//  1. Ingress: in the workspace namespace, allows sentinel → deployment on the deployment port
//  2. Egress: in the sentinel namespace, allows sentinel → deployment on the deployment port
//
// The control plane stores policies in the database so regional reconcilers can apply
// them without recomputing intent. The port is taken from the deployment's snapshotted
// settings so the policy always matches the running container port. Each deployment owns
// its own policies so that old deployments (serving traffic via SHA URLs) are not broken
// when a new deployment changes the port.
func (w *Workflow) ensureCiliumNetworkPolicy(
	ctx restate.WorkflowSharedContext,
	workspace db.Workspace,
	project db.FindProjectByIdRow,
	environment db.FindEnvironmentByIdRow,
	topologies []db.InsertDeploymentTopologyParams,
	deployment db.Deployment,
) error {

	existingPolicies, err := restate.Run(ctx, func(runCtx restate.RunContext) ([]db.CiliumNetworkPolicy, error) {
		return db.Query.FindCiliumNetworkPoliciesByDeploymentID(runCtx, w.db.RO(), deployment.ID)
	}, restate.WithName("find existing cilium policies"))

	if err != nil {
		return fmt.Errorf("failed to query existing cilium policies: %w", err)
	}

	// Key by region:k8s_name so we can distinguish ingress vs egress per region.
	existingByKey := make(map[string]db.CiliumNetworkPolicy)
	for _, policy := range existingPolicies {
		key := policy.Region + ":" + policy.K8sName
		existingByKey[key] = policy
	}

	for _, topo := range topologies {
		specs := buildPolicySpecs(workspace, environment, deployment)

		for _, spec := range specs {
			lookupKey := topo.Region + ":" + spec.k8sName
			existing, hasExisting := existingByKey[lookupKey]

			// Build the policy payload with the correct ID label before comparing.
			policyID := uid.New(uid.CiliumNetworkPolicyPrefix)
			if hasExisting {
				policyID = existing.ID
			}

			policyObj := spec.policy
			if policyObj.Labels == nil {
				policyObj.Labels = make(map[string]string)
			}
			policyObj.Labels[labels.LabelKeyNetworkPolicyID] = policyID

			policyPayload, err := json.Marshal(policyObj)
			if err != nil {
				return fmt.Errorf("failed to marshal cilium policy %s: %w", spec.k8sName, err)
			}

			// Skip the DB write if the policy is unchanged.
			if hasExisting && bytes.Equal(existing.Policy, policyPayload) {
				continue
			}

			policyVersion, err := hydrav1.NewVersioningServiceClient(ctx, topo.Region).NextVersion().Request(&hydrav1.NextVersionRequest{})
			if err != nil {
				return fmt.Errorf("failed to get next version for cilium policy %s: %w", spec.k8sName, err)
			}

			err = restate.RunVoid(ctx, func(runCtx restate.RunContext) error {

				if hasExisting {
					return db.Query.UpdateCiliumNetworkPolicyByEnvironmentRegionAndName(runCtx, w.db.RW(), db.UpdateCiliumNetworkPolicyByEnvironmentRegionAndNameParams{
						Policy:        policyPayload,
						Version:       policyVersion.GetVersion(),
						UpdatedAt:     sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
						EnvironmentID: environment.ID,
						Region:        topo.Region,
						K8sName:       spec.k8sName,
					})
				}

				err = db.Query.InsertCiliumNetworkPolicy(runCtx, w.db.RW(), db.InsertCiliumNetworkPolicyParams{
					ID:            policyID,
					WorkspaceID:   workspace.ID,
					ProjectID:     project.ID,
					EnvironmentID: environment.ID,
					DeploymentID:  deployment.ID,
					K8sName:       spec.k8sName,
					K8sNamespace:  spec.k8sNamespace,
					Region:        topo.Region,
					Policy:        policyPayload,
					Version:       policyVersion.GetVersion(),
					CreatedAt:     time.Now().UnixMilli(),
				})
				if err != nil && !db.IsDuplicateKeyError(err) {
					return fmt.Errorf("failed to insert cilium policy %s into db: %w", spec.k8sName, err)
				}
				return nil

			}, restate.WithName(fmt.Sprintf("upsert network policy %s", spec.k8sName)))
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// buildPolicySpecs returns the ingress and egress policy specs for a deployment.
func buildPolicySpecs(
	workspace db.Workspace,
	environment db.FindEnvironmentByIdRow,
	deployment db.Deployment,
) []ciliumPolicySpec {
	portStr := fmt.Sprintf("%d", deployment.Port)

	ingressName := fmt.Sprintf("sentinel-ingress-to-%s", deployment.K8sName)
	egressName := fmt.Sprintf("sentinel-egress-to-%s", deployment.K8sName)

	ingressLabels := labels.New().
		ManagedByKrane().
		ComponentDeployment().
		DeploymentID(deployment.ID).
		EnvironmentID(environment.ID).
		WorkspaceID(workspace.ID)

	egressLabels := labels.New().
		ManagedByKrane().
		ComponentSentinel().
		DeploymentID(deployment.ID).
		EnvironmentID(environment.ID).
		WorkspaceID(workspace.ID)

	//nolint:exhaustruct
	ingress := ciliumv2.CiliumNetworkPolicy{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "cilium.io/v2",
			Kind:       "CiliumNetworkPolicy",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      ingressName,
			Namespace: workspace.K8sNamespace.String,
			Labels:    ingressLabels,
		},
		Spec: &api.Rule{
			Description: fmt.Sprintf("Allow ingress from sentinel for workspace %s deployment %s", workspace.ID, deployment.ID),
			EndpointSelector: api.EndpointSelector{
				LabelSelector: &slim_metav1.LabelSelector{
					MatchLabels: labels.New().ManagedByKrane().ComponentDeployment().DeploymentID(deployment.ID),
				},
			},
			Ingress: []api.IngressRule{
				{
					IngressCommonRule: api.IngressCommonRule{
						FromEndpoints: []api.EndpointSelector{
							{
								LabelSelector: &slim_metav1.LabelSelector{
									MatchLabels: labels.New().Namespace(sentinelNamespace).WorkspaceID(workspace.ID).EnvironmentID(environment.ID),
								},
							},
						},
					},
					ToPorts: api.PortRules{
						{
							Ports: []api.PortProtocol{
								{
									Port:     portStr,
									Protocol: api.ProtoTCP,
								},
							},
						},
					},
				},
			},
		},
	}

	//nolint:exhaustruct
	egress := ciliumv2.CiliumNetworkPolicy{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "cilium.io/v2",
			Kind:       "CiliumNetworkPolicy",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      egressName,
			Namespace: sentinelNamespace,
			Labels:    egressLabels,
		},
		Spec: &api.Rule{
			Description: fmt.Sprintf("Allow sentinel egress to deployment %s for workspace %s", deployment.ID, workspace.ID),
			EndpointSelector: api.EndpointSelector{
				LabelSelector: &slim_metav1.LabelSelector{
					MatchLabels: labels.New().ComponentSentinel().EnvironmentID(environment.ID).WorkspaceID(workspace.ID),
				},
			},
			Egress: []api.EgressRule{
				{
					EgressCommonRule: api.EgressCommonRule{
						ToEndpoints: []api.EndpointSelector{
							{
								LabelSelector: &slim_metav1.LabelSelector{
									MatchLabels: labels.New().Namespace(workspace.K8sNamespace.String).ManagedByKrane().ComponentDeployment().DeploymentID(deployment.ID),
								},
							},
						},
					},
					ToPorts: api.PortRules{
						{
							Ports: []api.PortProtocol{
								{
									Port:     portStr,
									Protocol: api.ProtoTCP,
								},
							},
						},
					},
				},
			},
		},
	}

	return []ciliumPolicySpec{
		{k8sName: ingressName, k8sNamespace: workspace.K8sNamespace.String, policy: ingress},
		{k8sName: egressName, k8sNamespace: sentinelNamespace, policy: egress},
	}
}
