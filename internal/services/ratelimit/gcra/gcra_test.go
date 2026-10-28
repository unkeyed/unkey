package gcra

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
)

func TestModeOf(t *testing.T) {
	off := db.RatelimitNamespacesGlobalRatelimitModeOff
	shadow := db.RatelimitNamespacesGlobalRatelimitModeShadow
	on := db.RatelimitNamespacesGlobalRatelimitModeOn

	cases := []struct {
		name  string
		modes []db.RatelimitNamespacesGlobalRatelimitMode
		want  db.RatelimitNamespacesGlobalRatelimitMode
	}{
		{name: "all on", modes: []db.RatelimitNamespacesGlobalRatelimitMode{on, on}, want: on},
		{name: "shadow and on", modes: []db.RatelimitNamespacesGlobalRatelimitMode{on, shadow}, want: shadow},
		{name: "off after on", modes: []db.RatelimitNamespacesGlobalRatelimitMode{on, off}, want: off},
		{name: "off before shadow", modes: []db.RatelimitNamespacesGlobalRatelimitMode{off, shadow}, want: off},
		{name: "unknown value", modes: []db.RatelimitNamespacesGlobalRatelimitMode{on, ""}, want: off},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, ModeOf(tc.modes...))
		})
	}
}
