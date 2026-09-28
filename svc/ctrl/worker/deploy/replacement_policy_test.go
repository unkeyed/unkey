package deploy

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	privatecontract "github.com/unkeyed/unkey/pkg/privatenetwork"
)

func TestReplacementPolicy(t *testing.T) {
	tests := []struct {
		name         string
		kind         mysqltype.EnvironmentKind
		slug         string
		promoteLive  bool
		replaceAfter time.Duration
	}{
		{name: "built-in production becomes live", kind: mysqltype.EnvironmentKindProduction, slug: "production", promoteLive: true},
		{name: "production canary replaces canary siblings", kind: mysqltype.EnvironmentKindProduction, slug: "private-network", replaceAfter: privatecontract.ReplacementOverlap},
		{name: "preview replaces branch siblings", kind: mysqltype.EnvironmentKindPreview, slug: "preview", replaceAfter: privatecontract.ReplacementOverlap},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			promoteLive, replaceAfter := replacementPolicy(test.kind, test.slug)
			require.Equal(t, test.promoteLive, promoteLive)
			require.Equal(t, test.replaceAfter, replaceAfter)
		})
	}
}
