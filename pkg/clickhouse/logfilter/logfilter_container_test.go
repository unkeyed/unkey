package logfilter_test

import (
	"testing"
	"time"

	ch "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/stretchr/testify/require"

	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/clickhouse/logfilter"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
)

const (
	colPath   logfilter.Column = "path"
	colMethod logfilter.Column = "method"
	colStatus logfilter.Column = "response_status"
	colTime   logfilter.Column = "time"
)

// TestFiltersMatchLiterally runs the builder against a real server over both
// transports, because each one carries server-side parameters differently and
// the escaping in the builder has to be right for both.
func TestFiltersMatchLiterally(t *testing.T) {
	cfg := containers.ClickHouse(t)

	for name, dsn := range map[string]string{"native": cfg.DSN, "http": cfg.HTTPDSN} {
		t.Run(name, func(t *testing.T) {
			client, err := clickhouse.New(clickhouse.Config{URL: dsn})
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, client.Close()) })
			ctx := t.Context()

			workspaceID := uid.New(uid.WorkspacePrefix)
			otherWorkspaceID := uid.New(uid.WorkspacePrefix)
			paths := []string{
				"/100%", "/100x", "/a_b", "/axb", `C:\temp`, "C:\temp", "a\tb", "a\nb", "a\rb",
				"it's", "a\x00b", "a\xffb", "/KEBAP", "/kebap", "/v2/keys.verifyKey", "/v2/keys.createKey",
			}
			now := time.Now().UnixMilli()

			batch, err := client.Conn().PrepareBatch(ctx, clickhouse.InsertQuery[schema.ApiRequest]())
			require.NoError(t, err)
			for _, ws := range []string{workspaceID, otherWorkspaceID} {
				for i, path := range paths {
					status, method := int32(200), "POST"
					if i%2 == 1 {
						status, method = 404, "GET"
					}
					row := schema.ApiRequest{
						RequestID:       uid.New(uid.RequestPrefix),
						Time:            now - int64(i),
						WorkspaceID:     ws,
						Host:            "api.unkey.com",
						Method:          method,
						Path:            path,
						QueryString:     "",
						QueryParams:     map[string][]string{},
						RequestHeaders:  []string{},
						RequestBody:     "",
						ResponseStatus:  status,
						ResponseHeaders: []string{},
						ResponseBody:    "",
						Error:           "",
						ServiceLatency:  1,
						UserAgent:       "",
						IpAddress:       "",
						Region:          "test",
					}
					require.NoError(t, batch.AppendStruct(&row))
				}
			}
			require.NoError(t, batch.Send())

			hundreds := logfilter.Text{StartsWith: []string{"/100"}}
			tests := []struct {
				name   string
				text   logfilter.Text
				method logfilter.Exact
				status logfilter.Int
				time   logfilter.Int
				want   []string
			}{
				{name: "percent is literal", text: logfilter.Text{Contains: []string{"%"}}, want: []string{"/100%"}},
				{name: "underscore is literal", text: logfilter.Text{Contains: []string{"_"}}, want: []string{"/a_b"}},
				{name: "backslash round-trips", text: logfilter.Text{Is: []string{`C:\temp`}}, want: []string{`C:\temp`}},
				{name: "tab round-trips", text: logfilter.Text{Is: []string{"a\tb"}}, want: []string{"a\tb"}},
				{name: "line feed round-trips", text: logfilter.Text{Is: []string{"a\nb"}}, want: []string{"a\nb"}},
				{name: "carriage return round-trips", text: logfilter.Text{Is: []string{"a\rb"}}, want: []string{"a\rb"}},
				{name: "quote round-trips", text: logfilter.Text{Is: []string{"it's"}}, want: []string{"it's"}},
				{name: "null byte round-trips", text: logfilter.Text{Is: []string{"a\x00b"}}, want: []string{"a\x00b"}},
				{name: "invalid utf-8 round-trips", text: logfilter.Text{Is: []string{"a\xffb"}}, want: []string{"a\xffb"}},
				{name: "case-sensitive", text: logfilter.Text{EndsWith: []string{"KEBAP"}}, want: []string{"/KEBAP"}},
				{
					name: "positives OR and negatives AND NOT",
					text: logfilter.Text{
						StartsWith:  []string{"/v2/keys", "/100"},
						NotContains: []string{"verifyKey", "x"},
					},
					want: []string{"/100%", "/v2/keys.createKey"},
				},
				{
					name:   "fields AND",
					text:   logfilter.Text{StartsWith: []string{"/a"}},
					status: logfilter.Int{IsNot: []int64{200}},
					want:   []string{"/axb"},
				},
				{
					name: "is and contains OR on one field",
					text: logfilter.Text{Is: []string{"/axb"}, Contains: []string{"createKey"}},
					want: []string{"/axb", "/v2/keys.createKey"},
				},
				{
					name:   "exact on a low cardinality column",
					text:   logfilter.Text{StartsWith: []string{"/v2/"}},
					method: logfilter.Exact{Is: []string{"GET", "PUT"}},
					want:   []string{"/v2/keys.createKey"},
				},
				{
					name:   "range bounds AND",
					text:   logfilter.Text{StartsWith: []string{"/100"}},
					status: logfilter.Int{Gte: new(int64(400)), Lt: new(int64(500))},
					want:   []string{"/100x"},
				},
				{name: "gte includes the bound", text: hundreds, status: logfilter.Int{Gte: new(int64(404))}, want: []string{"/100x"}},
				{name: "gt excludes the bound", text: hundreds, status: logfilter.Int{Gt: new(int64(404))}, want: nil},
				{name: "lte includes the bound", text: hundreds, status: logfilter.Int{Lte: new(int64(404))}, want: []string{"/100%", "/100x"}},
				{name: "lt excludes the bound", text: hundreds, status: logfilter.Int{Lt: new(int64(404))}, want: []string{"/100%"}},
				{
					name: "time window",
					time: logfilter.Int{Gte: new(now - 3), Lt: new(now - 1)},
					want: []string{"/a_b", "/axb"},
				},
				{name: "range and is AND", status: logfilter.Int{Is: []int64{200}, Gt: new(int64(200))}, want: nil},
				{name: "out of range int does not wrap", status: logfilter.Int{Is: []int64{4294967700}}, want: nil},
				{name: "injection is a value", text: logfilter.Text{Is: []string{"' OR 1=1 --"}}, want: nil},
			}

			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					b := logfilter.New()
					b.Text(colPath, tt.text)
					b.Exact(colMethod, tt.method)
					b.Int(colStatus, tt.status)
					b.Int(colTime, tt.time)
					where, params := b.Build()
					params["workspace_id"] = workspaceID

					var got []string
					rows, err := client.Conn().Query(
						ch.Context(ctx, ch.WithParameters(params)),
						"SELECT path FROM default.api_requests_raw_v2 WHERE workspace_id = {workspace_id:String} AND "+where,
					)
					require.NoError(t, err)
					t.Cleanup(func() { require.NoError(t, rows.Close()) })
					for rows.Next() {
						var path string
						require.NoError(t, rows.Scan(&path))
						got = append(got, path)
					}
					require.NoError(t, rows.Err())

					require.ElementsMatch(t, tt.want, got)
				})
			}
		})
	}
}
