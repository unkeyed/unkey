package clickhouse

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConfigureUser_FailsClosedAtEveryProvisioningStep(t *testing.T) {
	config := securityTestUserConfig()
	const operationCount = 7

	for failAfter := 0; failAfter < operationCount; failAfter++ {
		t.Run(strings.Repeat("step_", failAfter+1), func(t *testing.T) {
			state := newProvisioningSecurityState(config.AllowedTables)
			calls := 0
			exec := func(_ context.Context, query string, _ ...any) error {
				calls++
				if calls == failAfter+1 {
					return errors.New("injected provisioning failure")
				}
				state.apply(query)
				// Security guarantee: reads stay denied after every partial provisioning operation.
				require.False(t, state.canReadWorkspaceAnalytics())
				return nil
			}

			err := configureUserWithExec(context.Background(), config, exec)
			require.ErrorContains(t, err, "injected provisioning failure")
			require.False(t, state.canReadWorkspaceAnalytics())

			calls = 0
			err = configureUserWithExec(context.Background(), config, exec)
			require.Error(t, err)
			// Security guarantee: retries fail closed at the same partial state.
			require.False(t, state.canReadWorkspaceAnalytics())
		})
	}
}

func TestConfigureUser_GrantsOnlyAfterAllConstraints(t *testing.T) {
	config := securityTestUserConfig()
	state := newProvisioningSecurityState(config.AllowedTables)

	err := configureUserWithExec(context.Background(), config, func(_ context.Context, query string, _ ...any) error {
		state.apply(query)
		if strings.HasPrefix(strings.TrimSpace(query), "GRANT SELECT") {
			// Security guarantee: the sole grant is last and exposes only fully constrained reads.
			require.True(t, state.hasAllPolicies())
			require.True(t, state.profileConstrained)
			require.True(t, state.quotaConstrained)
			require.Equal(t, 7, state.operations)
		}
		return nil
	})
	require.NoError(t, err)
	require.True(t, state.canReadWorkspaceAnalytics())
}

func TestConfigureUser_QuotaUpdatePreservesConstraintsAndRevokedGrants(t *testing.T) {
	config := securityTestUserConfig()
	state := newProvisioningSecurityState(config.AllowedTables)
	require.NoError(t, configureUserWithExec(context.Background(), config, state.exec))
	require.True(t, state.canReadWorkspaceAnalytics())

	config.AllowedTables = config.AllowedTables[:1]
	state.allowedTables = config.AllowedTables
	config.MaxQueriesPerWindow = 5
	require.NoError(t, configureUserWithExec(context.Background(), config, state.exec))

	// Security guarantee: limit updates retain immutable constraints and never restore removed grants.
	require.True(t, state.profileConstrained)
	require.True(t, state.quotaConstrained)
	require.Equal(t, config.AllowedTables, state.grantedTables)
	require.NotContains(t, state.grantedTables, "default.key_verifications_per_minute_v3")
}

type provisioningSecurityState struct {
	allowedTables      []string
	grantedTables      []string
	policies           map[string]bool
	profileConstrained bool
	quotaConstrained   bool
	operations         int
}

func newProvisioningSecurityState(tables []string) *provisioningSecurityState {
	return &provisioningSecurityState{allowedTables: tables, policies: make(map[string]bool)}
}

func (s *provisioningSecurityState) exec(_ context.Context, query string, _ ...any) error {
	s.apply(query)
	return nil
}

func (s *provisioningSecurityState) apply(query string) {
	s.operations++
	normalized := strings.Join(strings.Fields(query), " ")
	switch {
	case strings.HasPrefix(normalized, "REVOKE ALL"):
		s.grantedTables = nil
	case strings.HasPrefix(normalized, "CREATE ROW POLICY"):
		for _, table := range s.allowedTables {
			if strings.Contains(normalized, " ON "+table+" ") {
				s.policies[table] = true
			}
		}
	case strings.HasPrefix(normalized, "CREATE SETTINGS PROFILE"):
		s.profileConstrained = strings.Count(normalized, " CONST") == 4
	case strings.HasPrefix(normalized, "CREATE QUOTA"):
		s.quotaConstrained = strings.Contains(normalized, "MAX queries") && strings.Contains(normalized, "MAX execution_time")
	case strings.HasPrefix(normalized, "GRANT SELECT"):
		s.grantedTables = append([]string(nil), s.allowedTables...)
	}
}

func (s *provisioningSecurityState) hasAllPolicies() bool {
	for _, table := range s.allowedTables {
		if !s.policies[table] {
			return false
		}
	}
	return true
}

func (s *provisioningSecurityState) canReadWorkspaceAnalytics() bool {
	return len(s.grantedTables) == len(s.allowedTables) && s.hasAllPolicies() && s.profileConstrained && s.quotaConstrained
}

func securityTestUserConfig() UserConfig {
	return UserConfig{
		WorkspaceID:               "ws_security_test",
		Username:                  "ws_security_test",
		Password:                  "password",
		AllowedTables:             []string{"default.key_verifications_raw_v2", "default.key_verifications_per_minute_v3"},
		QuotaDurationSeconds:      60,
		MaxQueriesPerWindow:       10,
		MaxExecutionTimePerWindow: 30,
		MaxQueryExecutionTime:     5,
		MaxQueryMemoryBytes:       1024,
		MaxQueryResultRows:        100,
		RetentionDays:             7,
	}
}
