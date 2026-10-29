package discovery

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/cli"
	"github.com/unkeyed/unkey/pkg/uid"
)

func TestFlagMetadataDoesNotReadCurrentOrEnvironmentValues(t *testing.T) {
	secret := uid.New(uid.KeyPrefix)
	t.Setenv("UNKEY_DISCOVERY_TEST_SECRET", secret)
	for _, test := range []struct {
		flag cli.Flag
		kind string
	}{
		{cli.String("text", "Text", cli.EnvVar("UNKEY_DISCOVERY_TEST_SECRET")), "string"},
		{cli.Enum("choice", "Choice", []string{"first", "second"}, cli.Default("second")), "string"},
		{cli.Bool("enabled", "Enabled", cli.Default(true)), "boolean"},
		{cli.Int("count", "Count", cli.Default(42)), "integer"},
		{cli.Int64("count64", "Count", cli.Default(int64(84))), "integer"},
		{cli.Float("ratio", "Ratio", cli.Default(1.5)), "number"},
		{cli.StringSlice("names", "Names", cli.Default([]string{secret})), "string[]"},
		{cli.Duration("timeout", "Timeout", cli.Default(time.Minute)), "duration"},
	} {
		t.Run(test.flag.Name(), func(t *testing.T) {
			metadata, err := flagMetadata(test.flag)
			require.NoError(t, err)
			require.Equal(t, test.kind, metadata.Type)
			data, err := json.Marshal(metadata)
			require.NoError(t, err)
			require.NotContains(t, string(data), secret)
			require.NotContains(t, string(data), "default")
		})
	}
}
