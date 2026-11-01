package util

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/rivo/uniseg"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/sdks/api/go/v3/models/components"
	"github.com/unkeyed/unkey/pkg/cli"
	"github.com/unkeyed/unkey/pkg/uid"
	"golang.org/x/sys/unix"
)

func TestOutputWhoamiTreeWrapsWithoutLosingUnicode(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	name := strings.Repeat("分析Cafe\u0301", 9)
	permission := strings.Repeat("documents.read.", 7)
	response := &components.V2KeysWhoamiResponseBody{Data: components.KeyResponseData{
		KeyID: uid.New(uid.KeyPrefix), Name: &name, Start: "demo_", Enabled: true,
		Permissions: []string{permission}, CreatedAt: 1700000000000,
	}}
	for _, width := range []uint16{24, 44, 80} {
		t.Run(strconv.Itoa(int(width)), func(t *testing.T) {
			output := captureTerminalOutput(t, width, response)
			require.NotContains(t, output, "\x1b")
			var unwrapped strings.Builder
			for _, line := range strings.Split(output, "\n") {
				require.LessOrEqual(t, uniseg.StringWidth(line), int(width), line)
				unwrapped.WriteString(strings.TrimLeft(line, "│├└─ "))
			}
			for _, want := range []string{name, permission, response.Data.KeyID, "2023-11-14T22:13:20Z"} {
				require.Contains(t, unwrapped.String(), want)
			}
		})
	}
}

func TestOutputTerminalTablesFitWithoutLosingSelectors(t *testing.T) {
	name := "Payments for international customers"
	response := &components.V2ProjectsListProjectsResponseBody{
		Data: []components.Project{
			{ID: uid.New(uid.ProjectPrefix), Name: name, Slug: "payments", UpdatedAt: new(time.Now().Add(-150 * time.Minute).UnixMilli())},
			{ID: uid.New(uid.ProjectPrefix), Name: "分析", Slug: "analytics", UpdatedAt: new(time.Now().Add(150 * time.Minute).UnixMilli())},
			{ID: uid.New(uid.ProjectPrefix), Name: "Cafe\u0301", Slug: "cafe"},
		},
	}
	for _, width := range []uint16{80, 42, 24} {
		t.Run(strconv.Itoa(int(width)), func(t *testing.T) {
			output := captureTerminalOutput(t, width, response)
			require.Contains(t, output, "payments")
			require.Contains(t, output, "analytics")
			require.Contains(t, output, "Cafe\u0301")
			require.Contains(t, output, "2 hours ago")
			require.Contains(t, output, "in 2 hours")
			require.NotContains(t, output, response.Data[0].ID)
			if width == 24 {
				require.Contains(t, output, "Name: "+name)
				require.Contains(t, output, "Slug: payments")
				require.Contains(t, output, "Updated: -")
				return
			}
			lines := strings.Split(strings.TrimSpace(output), "\n")
			require.Len(t, lines, 4)
			require.Equal(t, []string{"NAME", "SLUG", "UPDATED"}, strings.Fields(lines[0]))
			for _, line := range lines {
				require.LessOrEqual(t, uniseg.StringWidth(line), int(width))
			}
			for i, slug := range []string{"payments", "analytics", "cafe"} {
				prefix, _, found := strings.Cut(lines[i+1], slug)
				require.True(t, found)
				header, _, _ := strings.Cut(lines[0], "SLUG")
				require.Equal(t, uniseg.StringWidth(header), uniseg.StringWidth(prefix))
			}
			if width == 42 {
				require.Contains(t, output, "…")
			} else {
				require.Contains(t, output, name)
			}
		})
	}
}

func TestOutputNarrowTerminalPreservesCredentialsAndJSON(t *testing.T) {
	plaintext := strings.Repeat("secret", 30)
	response := &components.V2ApisListKeysResponseBody{
		Meta: components.Meta{RequestID: uid.New(uid.RequestPrefix)},
		Data: []components.KeyResponseData{{KeyID: uid.New(uid.KeyPrefix), Name: new("Payments"), Plaintext: &plaintext, Enabled: true}},
	}
	output := captureTerminalOutput(t, 32, response)
	for _, want := range []string{"Name: Payments", "Enabled", response.Data[0].KeyID, plaintext} {
		require.Contains(t, output, want)
	}
	require.NotContains(t, output, "…")
	output = captureTerminalOutput(t, 32, response, "--json")
	var got components.V2ApisListKeysResponseBody
	require.NoError(t, json.Unmarshal([]byte(output), &got))
	require.Equal(t, *response, got)
}

func captureTerminalOutput(t *testing.T, width uint16, response any, args ...string) string {
	t.Helper()
	t.Setenv("UNKEY_OUTPUT", "")
	master, err := os.OpenFile("/dev/ptmx", os.O_RDWR|unix.O_NOCTTY, 0)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, master.Close()) })
	require.NoError(t, unix.IoctlSetPointerInt(int(master.Fd()), unix.TIOCSPTLCK, 0))
	number, err := unix.IoctlGetInt(int(master.Fd()), unix.TIOCGPTN)
	require.NoError(t, err)
	slave, err := os.OpenFile("/dev/pts/"+strconv.Itoa(number), os.O_RDWR|unix.O_NOCTTY, 0)
	require.NoError(t, err)
	t.Cleanup(func() {
		if err := slave.Close(); err != nil {
			require.ErrorIs(t, err, os.ErrClosed)
		}
	})
	require.NoError(t, unix.IoctlSetWinsize(int(slave.Fd()), unix.TIOCSWINSZ, &unix.Winsize{Col: width, Row: 24}))
	type result struct {
		data []byte
		err  error
	}
	read := make(chan result, 1)
	go func() {
		data, err := io.ReadAll(master)
		read <- result{data: data, err: err}
	}()
	original := os.Stdout
	os.Stdout = slave
	t.Cleanup(func() { os.Stdout = original })
	cmd := &cli.Command{
		Name: "test", Flags: []cli.Flag{OutputFlag(), cli.Bool("json", "JSON output")},
		Action: func(_ context.Context, cmd *cli.Command) error { return Output(cmd, response) },
	}
	err = cmd.Run(t.Context(), append([]string{"test"}, args...))
	os.Stdout = original
	require.NoError(t, slave.Close())
	got := <-read
	require.NoError(t, err)
	require.True(t, got.err == nil || errors.Is(got.err, unix.EIO), "read terminal: %v", got.err)
	return strings.ReplaceAll(string(got.data), "\r\n", "\n")
}
