package acme

import (
	"testing"

	"github.com/go-acme/lego/v4/lego"
	"github.com/stretchr/testify/require"
)

func TestNewLegoConfig(t *testing.T) {
	user := &AcmeUser{key: nil, WorkspaceID: "ws_test", EmailDomain: "unkey.dev", Registration: nil}

	t.Run("empty directory keeps lego default", func(t *testing.T) {
		require.Equal(t, lego.LEDirectoryProduction, newLegoConfig(user, "").CADirURL)
	})

	t.Run("directory overrides lego default", func(t *testing.T) {
		require.Equal(t, lego.LEDirectoryStaging, newLegoConfig(user, lego.LEDirectoryStaging).CADirURL)
	})
}
