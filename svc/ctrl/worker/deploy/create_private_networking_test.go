package deploy_test

import (
	"context"
	"sync"
	"testing"

	"github.com/open-feature/go-sdk/openfeature"
	"github.com/stretchr/testify/require"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	vaultv1 "github.com/unkeyed/unkey/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/gen/rpc/vault"
	"github.com/unkeyed/unkey/pkg/featureflag"
	"github.com/unkeyed/unkey/pkg/uid"
)

func TestCreatePrivateNetworkingDecision(t *testing.T) {
	ctx := context.Background()

	t.Run("a workspace without bindings is not evaluated", func(t *testing.T) {
		h := newCreateHarness(t, ctx)
		h.flags.set(h.orgID, 0, true)

		deploymentID := uid.New(uid.DeploymentPrefix)
		h.create(t, ctx, deploymentID, h.imageRequest())

		require.False(t, h.deployment(t, ctx, deploymentID).PrivateNetworking)
		require.Zero(t, h.flags.evaluations(h.orgID))
	})

	t.Run("an enabled flag is evaluated for the workspace organization and persisted", func(t *testing.T) {
		h := newCreateHarness(t, ctx)
		h.bindOtherApps(t, ctx)
		h.flags.set(h.orgID, 0, true)

		deploymentID := uid.New(uid.DeploymentPrefix)
		h.create(t, ctx, deploymentID, h.imageRequest())

		require.True(t, h.deployment(t, ctx, deploymentID).PrivateNetworking)
		require.Equal(t, 1, h.flags.evaluations(h.orgID))
		require.NotEqual(t, h.workspaceID, h.orgID, "team.id is the WorkOS organization, not the workspace")
	})

	t.Run("a disabled flag persists false and snapshots no binding variables", func(t *testing.T) {
		h := newCreateHarness(t, ctx)
		target := h.newApp(t, ctx)
		h.bind(t, ctx, h.appID, h.environmentID, target.appID)
		h.flags.set(h.orgID, 0, false)

		deploymentID := uid.New(uid.DeploymentPrefix)
		h.create(t, ctx, deploymentID, h.imageRequest())

		row := h.deployment(t, ctx, deploymentID)
		require.False(t, row.PrivateNetworking)
		require.NotContains(t, string(row.EncryptedEnvironmentVariables), "DATABASE_HOST")
		require.Equal(t, 1, h.flags.evaluations(h.orgID))
	})

	t.Run("a provider outage is retried until the flag answers", func(t *testing.T) {
		h := newCreateHarness(t, ctx)
		h.bindOtherApps(t, ctx)
		h.flags.set(h.orgID, 2, true)

		deploymentID := uid.New(uid.DeploymentPrefix)
		h.create(t, ctx, deploymentID, h.imageRequest())

		require.True(t, h.deployment(t, ctx, deploymentID).PrivateNetworking)
		require.Equal(t, 3, h.flags.evaluations(h.orgID))
		h.awaitDeploy(t, deploymentID)
	})

	t.Run("a persistent flag error fails the create before the row is written", func(t *testing.T) {
		h := newCreateHarness(t, ctx)
		h.bindOtherApps(t, ctx)
		h.flags.set(h.orgID, 1000, true)

		deploymentID := uid.New(uid.DeploymentPrefix)
		_, err := h.tryCreate(ctx, deploymentID, h.imageRequest())
		require.Error(t, err)

		require.Zero(t, h.countDeployments(t, ctx))
		require.Greater(t, h.flags.evaluations(h.orgID), 1, "the evaluation is retried before the create fails")
		h.requireNoDeploy(t, deploymentID)
	})

	t.Run("a skipped deployment is not evaluated", func(t *testing.T) {
		h := newCreateHarness(t, ctx)
		h.bindOtherApps(t, ctx)
		h.flags.set(h.orgID, 1000, true)

		deploymentID := uid.New(uid.DeploymentPrefix)
		req := h.imageRequest()
		req.Decision = hydrav1.CreateDecision_CREATE_DECISION_SKIP
		h.create(t, ctx, deploymentID, req)

		require.False(t, h.deployment(t, ctx, deploymentID).PrivateNetworking)
		require.Zero(t, h.flags.evaluations(h.orgID))
	})
}

func TestCreateApprovalReusesPrivateNetworkingDecision(t *testing.T) {
	ctx := context.Background()

	for _, test := range []struct {
		name     string
		failures int
		value    bool
	}{
		{name: "after the flag turns off", failures: 0, value: false},
		{name: "while the provider fails", failures: 1000, value: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			h := newCreateHarness(t, ctx)
			target := h.newApp(t, ctx)
			h.bind(t, ctx, h.appID, h.environmentID, target.appID)
			h.flags.set(h.orgID, 0, true)

			deploymentID := uid.New(uid.DeploymentPrefix)
			push := h.imageRequest()
			push.Decision = hydrav1.CreateDecision_CREATE_DECISION_AWAIT_APPROVAL
			h.create(t, ctx, deploymentID, push)

			pushed := h.deployment(t, ctx, deploymentID)
			require.True(t, pushed.PrivateNetworking)
			require.Contains(t, string(pushed.EncryptedEnvironmentVariables), "ciphertext-for-DATABASE_HOST")
			require.Equal(t, 1, h.flags.evaluations(h.orgID))

			h.flags.set(h.orgID, test.failures, test.value)
			h.approve(t, ctx, deploymentID)
			resp := h.create(t, ctx, deploymentID, h.imageRequest())
			require.Equal(t, hydrav1.CreateOutcome_CREATE_OUTCOME_CREATED, resp.GetOutcome())
			h.awaitDeploy(t, deploymentID)

			approved := h.deployment(t, ctx, deploymentID)
			require.True(t, approved.PrivateNetworking)
			require.Equal(t, pushed.EncryptedEnvironmentVariables, approved.EncryptedEnvironmentVariables)
			require.Zero(t, h.flags.evaluations(h.orgID), "an approval reuses the stored decision")
		})
	}

	t.Run("a rebuild under a new id evaluates the flag again", func(t *testing.T) {
		h := newCreateHarness(t, ctx)
		target := h.newApp(t, ctx)
		h.bind(t, ctx, h.appID, h.environmentID, target.appID)
		source := h.imageDeployment(t, ctx, 1)
		_, err := h.database.RW().ExecContext(ctx, `UPDATE deployments SET private_networking = TRUE WHERE id = ?`, source.ID)
		require.NoError(t, err)
		h.flags.set(h.orgID, 0, false)

		deploymentID := uid.New(uid.DeploymentPrefix)
		h.create(t, ctx, deploymentID, h.existingRequest(source.ID, false))

		rebuilt := h.deployment(t, ctx, deploymentID)
		require.False(t, rebuilt.PrivateNetworking)
		require.NotContains(t, string(rebuilt.EncryptedEnvironmentVariables), "DATABASE_HOST")
		require.Equal(t, 1, h.flags.evaluations(h.orgID))
	})
}

func (h *createHarness) approve(t *testing.T, ctx context.Context, deploymentID string) {
	t.Helper()
	result, err := h.database.RW().ExecContext(ctx,
		`UPDATE deployments SET status = 'pending' WHERE id = ? AND status = 'awaiting_approval'`, deploymentID)
	require.NoError(t, err)
	rows, err := result.RowsAffected()
	require.NoError(t, err)
	require.Equal(t, int64(1), rows)
}

type bindingVault struct {
	vault.VaultServiceClient
}

func (bindingVault) EncryptBulk(_ context.Context, req *vaultv1.EncryptBulkRequest) (*vaultv1.EncryptBulkResponse, error) {
	items := make(map[string]*vaultv1.EncryptBulkResponseItem, len(req.GetItems()))
	for key := range req.GetItems() {
		items[key] = &vaultv1.EncryptBulkResponseItem{Encrypted: "ciphertext-for-" + key}
	}
	return &vaultv1.EncryptBulkResponse{Items: items}, nil
}

func (h *createHarness) bindOtherApps(t *testing.T, ctx context.Context) {
	t.Helper()
	caller, target := h.newApp(t, ctx), h.newApp(t, ctx)
	h.bind(t, ctx, caller.appID, caller.environmentID, target.appID)
}

func (h *createHarness) bind(t *testing.T, ctx context.Context, callerAppID, callerEnvironmentID, targetAppID string) {
	t.Helper()
	_, err := h.database.RW().ExecContext(ctx,
		`INSERT INTO app_bindings (id,workspace_id,project_id,app_id,environment_id,resource_type,resource_id,name,selection_mode,created_at)
		VALUES (?,?,?,?,?,'app',?,'database','automatic',1)`,
		uid.New("binding"), h.workspaceID, h.projectID, callerAppID, callerEnvironmentID, targetAppID)
	require.NoError(t, err)
}

type teamFlags struct {
	mu    sync.Mutex
	teams map[string]*teamFlag
}

type teamFlag struct {
	failures    int
	value       bool
	evaluations int
}

func newTeamFlags() *teamFlags {
	return &teamFlags{mu: sync.Mutex{}, teams: map[string]*teamFlag{}}
}

func (f *teamFlags) set(team string, failures int, value bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.teams[team] = &teamFlag{failures: failures, value: value, evaluations: 0}
}

func (f *teamFlags) evaluations(team string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	if flag, ok := f.teams[team]; ok {
		return flag.evaluations
	}
	return 0
}

func (f *teamFlags) Metadata() openfeature.Metadata { return openfeature.Metadata{Name: "team-flags"} }
func (f *teamFlags) Hooks() []openfeature.Hook      { return nil }

func (f *teamFlags) BooleanEvaluation(_ context.Context, flag string, defaultValue bool, flatCtx openfeature.FlattenedContext) openfeature.BoolResolutionDetail {
	team, _ := flatCtx["team"].(map[string]any)
	id, _ := team["id"].(string)

	f.mu.Lock()
	defer f.mu.Unlock()
	state, ok := f.teams[id]
	if flag != featureflag.PrivateNetworking || !ok {
		return openfeature.BoolResolutionDetail{Value: defaultValue, ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
			ResolutionError: openfeature.NewFlagNotFoundResolutionError("unknown flag or team"),
			Reason:          openfeature.ErrorReason,
		}}
	}
	state.evaluations++
	if state.failures > 0 {
		state.failures--
		return openfeature.BoolResolutionDetail{Value: defaultValue, ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
			ResolutionError: openfeature.NewProviderNotReadyResolutionError("outage"),
			Reason:          openfeature.ErrorReason,
		}}
	}
	return openfeature.BoolResolutionDetail{Value: state.value, ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
		Reason: openfeature.TargetingMatchReason,
	}}
}

func (f *teamFlags) StringEvaluation(_ context.Context, _ string, defaultValue string, _ openfeature.FlattenedContext) openfeature.StringResolutionDetail {
	return openfeature.StringResolutionDetail{Value: defaultValue, ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
		ResolutionError: openfeature.NewTypeMismatchResolutionError("boolean flags only"),
		Reason:          openfeature.ErrorReason,
	}}
}

func (f *teamFlags) FloatEvaluation(_ context.Context, _ string, defaultValue float64, _ openfeature.FlattenedContext) openfeature.FloatResolutionDetail {
	return openfeature.FloatResolutionDetail{Value: defaultValue, ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
		ResolutionError: openfeature.NewTypeMismatchResolutionError("boolean flags only"),
		Reason:          openfeature.ErrorReason,
	}}
}

func (f *teamFlags) IntEvaluation(_ context.Context, _ string, defaultValue int64, _ openfeature.FlattenedContext) openfeature.IntResolutionDetail {
	return openfeature.IntResolutionDetail{Value: defaultValue, ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
		ResolutionError: openfeature.NewTypeMismatchResolutionError("boolean flags only"),
		Reason:          openfeature.ErrorReason,
	}}
}

func (f *teamFlags) ObjectEvaluation(_ context.Context, _ string, defaultValue any, _ openfeature.FlattenedContext) openfeature.InterfaceResolutionDetail {
	return openfeature.InterfaceResolutionDetail{Value: defaultValue, ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
		ResolutionError: openfeature.NewTypeMismatchResolutionError("boolean flags only"),
		Reason:          openfeature.ErrorReason,
	}}
}
