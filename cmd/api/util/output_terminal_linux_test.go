package util

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"regexp"
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
	t.Setenv("NO_COLOR", "1")
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
	t.Setenv("NO_COLOR", "1")
	plaintext := strings.Repeat("secret", 30)
	response := &components.V2ApisListKeysResponseBody{
		Meta: components.Meta{RequestID: uid.New(uid.RequestPrefix)},
		Data: []components.KeyResponseData{{KeyID: uid.New(uid.KeyPrefix), Name: new("Payments"), Plaintext: &plaintext, Enabled: true}},
	}
	output := captureTerminalOutput(t, 32, response)
	for _, want := range []string{"Name: Payments", "Enabled", "Last Used: -", response.Data[0].KeyID, plaintext} {
		require.Contains(t, output, want)
	}
	require.NotContains(t, output, "…")
	output = captureTerminalOutput(t, 32, response, "--json")
	var got components.V2ApisListKeysResponseBody
	require.NoError(t, json.Unmarshal([]byte(output), &got))
	require.Equal(t, *response, got)
	for _, details := range []any{
		&components.V2KeysWhoamiResponseBody{Data: response.Data[0]},
		&components.V2KeysCreateKeyResponseBody{Data: components.V2KeysCreateKeyResponseData{KeyID: response.Data[0].KeyID, Key: plaintext}},
	} {
		output := captureTerminalOutput(t, 32, details)
		require.Contains(t, output, plaintext)
		require.NotContains(t, output, "…")
		output = captureTerminalOutput(t, 32, details, "--json")
		encoded, err := json.MarshalIndent(details, "", "  ")
		require.NoError(t, err)
		require.Equal(t, string(encoded)+"\n", output)
	}
}

func TestOutputTerminalTableStylesPreserveAlignmentAndTruncation(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	t.Setenv("CLICOLOR", "1")
	t.Setenv("TERM", "xterm-256color")
	response := &components.V2ProjectsListProjectsResponseBody{Data: []components.Project{
		{ID: uid.New(uid.ProjectPrefix), Name: "Payments for international customers", Slug: "payments"},
		{ID: uid.New(uid.ProjectPrefix), Name: "分析Cafe\u0301", Slug: "analytics"},
	}}
	styles := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	for _, width := range []uint16{80, 32, 24} {
		t.Run(strconv.Itoa(int(width)), func(t *testing.T) {
			colored := captureTerminalOutput(t, width, response)
			require.Contains(t, colored, "分析Cafe\u0301")
			require.NotContains(t, colored, "\x1b[1m")
			require.Contains(t, colored, "\x1b[90m-\x1b[0m")
			t.Setenv("NO_COLOR", "1")
			plain := captureTerminalOutput(t, width, response)
			require.NotContains(t, plain, "\x1b")
			require.Equal(t, plain, styles.ReplaceAllString(colored, ""))
			if width == 24 {
				require.Contains(t, plain, "Name: Payments for international customers")
				return
			}
			lines := strings.Split(strings.TrimSuffix(plain, "\n"), "\n")
			require.Len(t, lines, 3)
			for _, line := range lines {
				require.LessOrEqual(t, uniseg.StringWidth(line), int(width))
				require.Equal(t, strings.TrimRight(line, " "), line)
			}
			if width == 32 {
				require.Contains(t, colored, "…  payments")
			}
		})
	}
}

func TestOutputTerminalDomainTableStylesKeepLabelsAndPagination(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	t.Setenv("CLICOLOR", "1")
	t.Setenv("TERM", "xterm-256color")
	cursor := "next-page"
	response := &components.V2DomainsListDomainsResponseBody{
		Data: []components.Domain{
			{ID: uid.New(uid.DomainPrefix), Domain: "a.example", Status: components.DomainStatusVerified},
			{ID: uid.New(uid.DomainPrefix), Domain: "b.example", Status: components.DomainStatusPending},
			{ID: uid.New(uid.DomainPrefix), Domain: "c.example", Status: components.DomainStatusVerifying},
			{ID: uid.New(uid.DomainPrefix), Domain: "d.example", Status: components.DomainStatusFailed},
		},
		Pagination: components.Pagination{Cursor: &cursor, HasMore: true},
	}
	colored := captureTerminalOutput(t, 80, response)
	for _, styled := range []string{
		"\x1b[90mDOMAIN\x1b[0m", "\x1b[90mSTATUS\x1b[0m", "\x1b[90mCREATED\x1b[0m",
		"\x1b[32mverified\x1b[0m",
		"\x1b[33mpending\x1b[0m", "\x1b[33mverifying\x1b[0m",
		"\x1b[31mfailed\x1b[0m",
	} {
		require.Contains(t, colored, styled)
	}
	require.NotContains(t, colored, "\x1b[1m")
	t.Setenv("NO_COLOR", "1")
	plain := captureTerminalOutput(t, 80, response)
	require.Equal(t, "DOMAIN     STATUS     CREATED\n"+
		"a.example  verified   -\n"+
		"b.example  pending    -\n"+
		"c.example  verifying  -\n"+
		"d.example  failed     -\n"+
		"\nMore results available. Repeat this command with the same filters and --cursor='next-page'\n", plain)
	require.Equal(t, plain, regexp.MustCompile(`\x1b\[[0-9;]*m`).ReplaceAllString(colored, ""))
	t.Setenv("NO_COLOR", "")
	for _, flag := range []string{"--json", "--output=json"} {
		output := captureTerminalOutput(t, 80, response, flag)
		require.NotContains(t, output, "\x1b")
		var got components.V2DomainsListDomainsResponseBody
		require.NoError(t, json.Unmarshal([]byte(output), &got))
		require.Equal(t, *response, got)
	}
	output := captureOutput(t, response)
	require.NotContains(t, output, "\x1b")
	require.True(t, strings.HasPrefix(output, "DOMAIN     STATUS     CREATED\n"), output)
}

func TestOutputTerminalDeploymentTableHighlightsReady(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	t.Setenv("CLICOLOR", "1")
	t.Setenv("TERM", "xterm-256color")
	response := &components.V2DeploymentsListDeploymentsResponseBody{Data: []components.Deployment{
		{ID: uid.New(uid.DeploymentPrefix), Project: "demo", App: "api", Environment: "production", Status: "ready"},
	}}
	output := captureTerminalOutput(t, 160, response)
	require.Contains(t, output, "\x1b[32mready\x1b[0m")
	require.Len(t, strings.Split(strings.TrimSuffix(output, "\n"), "\n"), 2)
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
		Action: func(ctx context.Context, cmd *cli.Command) error { return Output(ctx, cmd, response) },
	}
	err = cmd.Run(t.Context(), append([]string{"test"}, args...))
	os.Stdout = original
	require.NoError(t, slave.Close())
	got := <-read
	require.NoError(t, err)
	require.True(t, got.err == nil || errors.Is(got.err, unix.EIO), "read terminal: %v", got.err)
	return strings.ReplaceAll(string(got.data), "\r\n", "\n")
}
