package deploy

import (
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

const (
	// deploymentPort is the port exposed by customer deployments for sentinel traffic.
	deploymentPort = 8080
)

// ensureCiliumNetworkPolicy persists missing Cilium policies for all deployment regions.
//
// The control plane stores policies in the database so regional reconcilers can apply
// them without recomputing intent. Policies are queried once per deployment to avoid
// per-region reads while still skipping regions that already have policy records.
func (w *Workflow) ensureCiliumNetworkPolicy(
	ctx restate.WorkflowSharedContext,
	workspace db.Workspace,
	project db.FindProjectByIdRow,
	environment db.FindEnvironmentByIdRow,
	topologies []db.InsertDeploymentTopologyParams,
) error {

	existingPolicies, err := restate.Run(ctx, func(runCtx restate.RunContext) ([]db.CiliumNetworkPolicy, error) {
		return db.Query.FindCiliumNetworkPoliciesByEnvironmentID(runCtx, w.db.RO(), environment.ID)
	}, restate.WithName("find existing cilium policies"))

	if err != nil {
		return fmt.Errorf("failed to query existing cilium policies: %w", err)
	}

	policyByRegion := make(map[string]db.CiliumNetworkPolicy)
	for _, policy := range existingPolicies {
		policyByRegion[policy.Region] = policy
	}

	for _, topo := range topologies {
		if _, ok := policyByRegion[topo.Region]; ok {
			continue
		}

		err = restate.RunVoid(ctx, func(runCtx restate.RunContext) error {

			policyVersion, err := hydrav1.NewVersioningServiceClient(ctx, topo.Region).NextVersion().Request(&hydrav1.NextVersionRequest{})
			if err != nil {
				return fmt.Errorf("failed to get next version for cilium policy: %w", err)
			}

			policyID := uid.New(uid.CiliumNetworkPolicyPrefix)
			policyLabels := labels.New().
				ManagedByKrane().
				ComponentDeployment().
				NetworkPolicyID(policyID).
				EnvironmentID(environment.ID).
				WorkspaceID(workspace.ID)
			//nolint:exhaustruct
			policy := ciliumv2.CiliumNetworkPolicy{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "cilium.io/v2",
					Kind:       "CiliumNetworkPolicy",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("sentinel-to-deployment-in-%s", environment.Slug),
					Namespace: workspace.K8sNamespace.String,
					Labels:    policyLabels,
				},
				Spec: &api.Rule{
					Description: fmt.Sprintf("Allow ingress from sentinel for workspace %s environment %s", workspace.ID, environment.ID),
					EndpointSelector: api.EndpointSelector{
						LabelSelector: &slim_metav1.LabelSelector{
							MatchLabels: labels.New().ManagedByKrane().ComponentDeployment().EnvironmentID(environment.ID),
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
											Port:     fmt.Sprintf("%d", deploymentPort),
											Protocol: api.ProtoTCP,
										},
									},
								},
							},
						},
					},
				},
			}

			policyPayload, err := json.Marshal(policy)
			if err != nil {
				return fmt.Errorf("failed to marshal cilium policy: %w", err)
			}
			err = db.Query.InsertCiliumNetworkPolicy(runCtx, w.db.RW(), db.InsertCiliumNetworkPolicyParams{
				ID:            policyID,
				WorkspaceID:   workspace.ID,
				ProjectID:     project.ID,
				EnvironmentID: environment.ID,
				K8sName:       policy.GetName(),
				Region:        topo.Region,
				Policy:        policyPayload,
				Version:       policyVersion.GetVersion(),
				CreatedAt:     time.Now().UnixMilli(),
			})
			if err != nil && !db.IsDuplicateKeyError(err) {
				return fmt.Errorf("failed to insert cilium policy into db: %w", err)
			}
			return nil

		}, restate.WithName("create network policy"))
		if err != nil {
			return err
		}

	}
	return nil
}
