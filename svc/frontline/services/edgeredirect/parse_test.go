package edgeredirect

import (
	"strings"
	"testing"

	edgeredirectv1 "github.com/unkeyed/unkey/gen/proto/frontline/edgeredirect/v1"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestParseConfig_EmptyInput(t *testing.T) {
	t.Parallel()
	cases := [][]byte{nil, {}, []byte("{}")}
	for _, raw := range cases {
		raw := raw
		t.Run(string(raw), func(t *testing.T) {
			t.Parallel()
			rules, err := ParseConfig(raw)
			if err != nil {
				t.Fatalf("err = %v", err)
			}
			if rules != nil {
				t.Fatalf("rules = %v, want nil", rules)
			}
		})
	}
}

func TestParseConfig_RoundTrip(t *testing.T) {
	t.Parallel()
	cfg := &edgeredirectv1.Config{
		Rules: []*edgeredirectv1.Rule{
			{
				Id:      "default-https",
				Enabled: true,
				Kind:    &edgeredirectv1.Rule_RequireHttps{RequireHttps: &edgeredirectv1.RequireHTTPS{}},
			},
			{
				Id:      "strip-www",
				Enabled: true,
				Status:  301,
				Kind:    &edgeredirectv1.Rule_StripWww{StripWww: &edgeredirectv1.StripWWW{}},
			},
			{
				Id:      "rewrite",
				Enabled: false,
				Kind: &edgeredirectv1.Rule_HostRewrite{HostRewrite: &edgeredirectv1.HostRewrite{
					From: "old.example.com",
					To:   "new.example.com",
				}},
			},
		},
	}
	raw, err := protojson.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	rules, err := ParseConfig(raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(rules) != 3 {
		t.Fatalf("rule count = %d, want 3", len(rules))
	}
	if rules[0].GetId() != "default-https" || !rules[0].GetEnabled() {
		t.Fatalf("rule[0] = %+v", rules[0])
	}
	if rules[1].GetStatus() != 301 {
		t.Fatalf("rule[1] status = %d, want 301", rules[1].GetStatus())
	}
	if rules[2].GetEnabled() {
		t.Fatalf("rule[2] should be disabled")
	}
	if got := rules[2].GetHostRewrite(); got == nil || got.GetFrom() != "old.example.com" {
		t.Fatalf("rule[2] host rewrite = %+v", got)
	}
}

func TestParseConfig_EmptyRulesArray(t *testing.T) {
	t.Parallel()
	rules, err := ParseConfig([]byte(`{"rules":[]}`))
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if rules != nil {
		t.Fatalf("rules = %v, want nil", rules)
	}
}

func TestParseConfig_Malformed(t *testing.T) {
	t.Parallel()
	_, err := ParseConfig([]byte("not json"))
	if err == nil {
		t.Fatal("expected error for malformed input")
	}
	if !strings.Contains(err.Error(), "unable to unmarshal") {
		t.Fatalf("err = %v, want internal-message about unmarshal", err)
	}
}

func TestParseConfig_DiscardsUnknownFields(t *testing.T) {
	t.Parallel()
	// Forward compat: a field not in this version of the proto should not
	// fail the whole parse.
	rules, err := ParseConfig([]byte(`{"rules":[{"id":"x","enabled":true,"requireHttps":{},"unknownFutureField":42}]}`))
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if len(rules) != 1 || rules[0].GetId() != "x" {
		t.Fatalf("rules = %+v", rules)
	}
}
