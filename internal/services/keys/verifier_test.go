package keys_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/internal/services/keys"
	keysdb "github.com/unkeyed/unkey/internal/services/keys/db"
)

func TestKeyVerifier_RejectsKeyspaceOutsideAllowlist(t *testing.T) {
	t.Parallel()

	verifier := &keys.KeyVerifier{
		Key:    keysdb.FindKeyForVerificationRow{KeyAuthID: "ks_actual"},
		Status: keys.StatusValid,
	}

	err := verifier.Verify(t.Context(), keys.WithKeyspaces("ks_other", "ks_another"))
	require.NoError(t, err)
	require.Equal(t, keys.StatusNotFound, verifier.Status)
}

func TestKeyVerifier_KeyspaceAllowlist(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name   string
		opts   []keys.VerifyOption
		status keys.KeyStatus
	}{
		{name: "omitted allows any keyspace", status: keys.StatusValid},
		{name: "matches second keyspace", opts: []keys.VerifyOption{keys.WithKeyspaces("ks_other", "ks_actual")}, status: keys.StatusValid},
		{name: "empty rejects every keyspace", opts: []keys.VerifyOption{keys.WithKeyspaces()}, status: keys.StatusNotFound},
		{name: "nil slice rejects every keyspace", opts: []keys.VerifyOption{keys.WithKeyspaces([]string(nil)...)}, status: keys.StatusNotFound},
		{name: "prefix does not match", opts: []keys.VerifyOption{keys.WithKeyspaces("ks_actual_suffix")}, status: keys.StatusNotFound},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			verifier := &keys.KeyVerifier{
				Key:    keysdb.FindKeyForVerificationRow{KeyAuthID: "ks_actual"},
				Status: keys.StatusValid,
			}

			err := verifier.Verify(t.Context(), tt.opts...)
			require.NoError(t, err)
			require.Equal(t, tt.status, verifier.Status)
		})
	}
}
