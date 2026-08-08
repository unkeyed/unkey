package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/unkeyed/sdks/api/go/v2/models/components"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
)

type policyUpdate struct {
	Name      json.RawMessage `json:"name"`
	Enabled   json.RawMessage `json:"enabled"`
	Match     json.RawMessage `json:"match"`
	Keyauth   json.RawMessage `json:"keyauth"`
	Ratelimit json.RawMessage `json:"ratelimit"`
	Firewall  json.RawMessage `json:"firewall"`
	Openapi   json.RawMessage `json:"openapi"`
}

func updatePolicyCmd() *cli.Command {
	return &cli.Command{
		Name: "update-policy", Usage: "Update a single policy in place without resending the environment's full policy list",
		Description: `Update a single policy in place without resending the environment's full policy list. The policy keeps its id and position, and all other policies are untouched.

Pass the policy fields to update as one JSON object. Omitted fields keep their stored values. Setting match to null or an empty array removes all match expressions. At least one update field is required, and at most one rule field may be set.

Required Permissions
- environment.*.update_policy (for any environment)
- environment.<environment_id>.update_policy (for a specific environment)

For full documentation, see https://www.unkey.com/docs/api-reference/gateway/update-policy` + util.Disclaimer,
		Examples: []string{`unkey api gateway update-policy --project=payments --app=payments-api --environment=production --policy-id=pol_123 --policy='{"enabled":false}'`, `unkey api gateway update-policy --project=payments --app=payments-api --environment=production --policy-id=pol_123 --policy='{"match":null}'`, `unkey api gateway update-policy --project=payments --app=payments-api --environment=production --policy-id=pol_123 --policy='{"firewall":{"action":"ACTION_DENY"}}'`},
		Flags:    []cli.Flag{cli.String("body", "Decode this JSON as the endpoint request body. Request-building flags are mutually exclusive."), util.RootKeyFlag(), util.APIURLFlag(), util.ConfigFlag(), util.OutputFlag(), cli.String("project", "Project ID or slug.", cli.Required(), cli.MutuallyExclusive("body")), cli.String("app", "App ID or slug.", cli.Required(), cli.MutuallyExclusive("body")), cli.String("environment", "Environment ID or slug.", cli.Required(), cli.MutuallyExclusive("body")), cli.String("policy-id", "ID of the policy to update.", cli.Required(), cli.MutuallyExclusive("body")), cli.String("policy", "Policy fields to update as a JSON object.", cli.Required(), cli.MutuallyExclusive("body"))},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			client, err := util.CreateClient(cmd)
			if err != nil {
				return err
			}

			if cmd.FlagIsSet("body") {
				body := cmd.String("body")
				body, err = normalizeUpdatePolicyBody(body)
				if err != nil {
					return err
				}
				res, err := util.SendBody(ctx, client.Gateway.UpdatePolicy, body)
				if err != nil {
					return err
				}
				return util.Output(cmd, res.V2GatewayUpdatePolicyResponseBody)
			}
			send := func(req components.V2GatewayUpdatePolicyRequestBody) error {
				res, err := client.Gateway.UpdatePolicy(ctx, req)
				if err != nil {
					return fmt.Errorf("%s", util.FormatError(err))
				}
				return util.Output(cmd, res.V2GatewayUpdatePolicyResponseBody)
			}
			var update policyUpdate
			decoder := json.NewDecoder(strings.NewReader(cmd.String("policy")))
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&update); err != nil {
				return fmt.Errorf("invalid JSON for --policy: %w", err)
			}
			if err := decoder.Decode(&struct{}{}); err != io.EOF {
				if err == nil {
					return fmt.Errorf("invalid JSON for --policy: multiple JSON values")
				}
				return fmt.Errorf("invalid JSON for --policy: %w", err)
			}

			updates := 0
			var name *string
			if update.Name != nil {
				if strings.TrimSpace(string(update.Name)) == "null" {
					return fmt.Errorf("name in --policy must be a string, not null")
				}
				if err := json.Unmarshal(update.Name, &name); err != nil {
					return fmt.Errorf("invalid name in --policy: %w", err)
				}
				updates++
			}
			var enabled *bool
			if update.Enabled != nil {
				if strings.TrimSpace(string(update.Enabled)) == "null" {
					return fmt.Errorf("enabled in --policy must be a boolean, not null")
				}
				if err := json.Unmarshal(update.Enabled, &enabled); err != nil {
					return fmt.Errorf("invalid enabled in --policy: %w", err)
				}
				updates++
			}
			if update.Match != nil {
				updates++
			}

			type ruleUpdate struct {
				name   string
				value  json.RawMessage
				target any
			}
			req := components.V2GatewayUpdatePolicyRequestBody{Project: cmd.String("project"), App: cmd.String("app"), Environment: cmd.String("environment"), PolicyID: cmd.String("policy-id"), Name: name, Enabled: enabled, Match: nil, Keyauth: nil, Ratelimit: nil, Firewall: nil, Openapi: nil}
			ruleUpdates := []ruleUpdate{
				{name: "keyauth", value: update.Keyauth, target: &req.Keyauth},
				{name: "ratelimit", value: update.Ratelimit, target: &req.Ratelimit},
				{name: "firewall", value: update.Firewall, target: &req.Firewall},
				{name: "openapi", value: update.Openapi, target: &req.Openapi},
			}
			rules := 0
			for _, rule := range ruleUpdates {
				if rule.value != nil {
					updates++
					rules++
				}
			}
			if updates == 0 {
				return fmt.Errorf("--policy must contain at least one update field")
			}
			if rules > 1 {
				return fmt.Errorf("--policy may contain at most one of keyauth, ratelimit, firewall, or openapi")
			}
			if update.Match != nil {
				if strings.TrimSpace(string(update.Match)) == "null" {
					req.Match = make([]components.MatchExpr, 0)
				} else if err := json.Unmarshal(update.Match, &req.Match); err != nil {
					return fmt.Errorf("invalid match in --policy: %w", err)
				}
			}
			for _, rule := range ruleUpdates {
				if rule.value != nil {
					if strings.TrimSpace(string(rule.value)) == "null" {
						return fmt.Errorf("%s in --policy must be a JSON object, not null", rule.name)
					}
					if err := json.Unmarshal(rule.value, rule.target); err != nil {
						return fmt.Errorf("invalid %s in --policy: %w", rule.name, err)
					}
				}
			}
			return send(req)
		},
	}
}

// normalizeUpdatePolicyBody preserves the API's explicit match:null clear
// operation across the generated SDK's decode and re-encode cycle.
func normalizeUpdatePolicyBody(body string) (string, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal([]byte(body), &fields); err != nil {
		return body, nil
	}
	match, ok := fields["match"]
	if !ok || strings.TrimSpace(string(match)) != "null" {
		return body, nil
	}
	fields["match"] = json.RawMessage("[]")
	normalized, err := json.Marshal(fields)
	if err != nil {
		return "", fmt.Errorf("normalize match in --body: %w", err)
	}
	return string(normalized), nil
}
