package api

import (
	"slices"
	"strings"

	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
)

// ResourceCommands shares endpoint handlers with [Cmd], without sharing parser state.
func ResourceCommands() []*cli.Command {
	root := &cli.Command{}
	for _, group := range groups() {
		for _, command := range group.Commands {
			legacy := group.Name + " " + command.Name
			path, ok := resourcePaths[legacy]
			if !ok {
				path = legacy
			}
			command.Usage = strings.ToUpper(command.Name[:1]) + strings.ReplaceAll(command.Name[1:], "-", " ")
			command.Description = strings.TrimSuffix(command.Description, util.Disclaimer)
			for i, example := range command.Examples {
				command.Examples[i] = strings.ReplaceAll(example, "unkey api "+legacy, "unkey "+path)
			}
			parts := strings.Fields(path)
			parent := root
			for _, name := range parts[:len(parts)-1] {
				var child *cli.Command
				for _, candidate := range parent.Commands {
					if candidate.Name == name {
						child = candidate
						break
					}
				}
				if child == nil {
					child = &cli.Command{
						Name:  name,
						Usage: "Manage " + name,
					}
					parent.Commands = append(parent.Commands, child)
				}
				if name == group.Name || (name == "portals" && group.Name == "portal") {
					child.Usage = group.Usage
				}
				parent = child
			}
			command.Name = parts[len(parts)-1]
			parent.Commands = append(parent.Commands, command)
		}
	}
	slices.SortFunc(root.Commands, func(a, b *cli.Command) int { return strings.Compare(a.Name, b.Name) })
	return root.Commands
}

var resourcePaths = map[string]string{
	"analytics get-gateway-requests":            "analytics gateway-requests",
	"analytics get-ratelimits":                  "analytics ratelimits",
	"analytics get-runtime-logs":                "analytics runtime-logs",
	"analytics get-verifications":               "analytics verifications",
	"apis create-api":                           "apis create",
	"apis delete-api":                           "apis delete",
	"apis get-api":                              "apis get",
	"apis list-keys":                            "keys list",
	"apps create-app":                           "apps create",
	"apps delete-app":                           "apps delete",
	"apps get-app":                              "apps get",
	"apps list-apps":                            "apps list",
	"apps update-app":                           "apps update",
	"deployments create-deployment":             "deployments create",
	"deployments get-deployment":                "deployments get",
	"deployments list-deployments":              "deployments list",
	"deployments promote-deployment":            "deployments promote",
	"deployments rollback-deployment":           "deployments rollback",
	"deployments start-deployment":              "deployments start",
	"deployments stop-deployment":               "deployments stop",
	"domains create-domain":                     "domains create",
	"domains delete-domain":                     "domains delete",
	"domains get-domain":                        "domains get",
	"domains list-domains":                      "domains list",
	"domains verify-domain":                     "domains verify",
	"environments get-environment":              "environments get",
	"environments list-environments":            "environments list",
	"environments list-environment-variables":   "environments variables list",
	"environments set-environment-variables":    "environments variables set",
	"environments remove-environment-variables": "environments variables remove",
	"environments update-settings":              "environments settings update",
	"gateway list-policies":                     "gateway policies list",
	"gateway set-policies":                      "gateway policies set",
	"gateway update-policy":                     "gateway policies update",
	"github install-app":                        "github install",
	"identities create-identity":                "identities create",
	"identities delete-identity":                "identities delete",
	"identities get-identity":                   "identities get",
	"identities list-identities":                "identities list",
	"identities update-identity":                "identities update",
	"keys create-key":                           "keys create",
	"keys delete-key":                           "keys delete",
	"keys get-key":                              "keys get",
	"keys verify-key":                           "keys verify",
	"keys update-key":                           "keys update",
	"keys reroll-key":                           "keys reroll",
	"keys whoami":                               "keys whoami",
	"keys migrate-keys":                         "keys migrate",
	"keys add-permissions":                      "keys permissions add",
	"keys remove-permissions":                   "keys permissions remove",
	"keys set-permissions":                      "keys permissions set",
	"keys add-roles":                            "keys roles add",
	"keys remove-roles":                         "keys roles remove",
	"keys set-roles":                            "keys roles set",
	"keys update-credits":                       "keys credits update",
	"permissions create-permission":             "permissions create",
	"permissions delete-permission":             "permissions delete",
	"permissions get-permission":                "permissions get",
	"permissions list-permissions":              "permissions list",
	"permissions create-role":                   "roles create",
	"permissions delete-role":                   "roles delete",
	"permissions get-role":                      "roles get",
	"permissions list-roles":                    "roles list",
	"permissions set-role-permissions":          "roles permissions set",
	"portal create-portal":                      "portals create",
	"portal delete-portal":                      "portals delete",
	"portal get-portal":                         "portals get",
	"portal update-portal":                      "portals update",
	"portal create-session":                     "portals sessions create",
	"projects create-project":                   "projects create",
	"projects delete-project":                   "projects delete",
	"projects get-project":                      "projects get",
	"projects list-projects":                    "projects list",
	"projects update-project":                   "projects update",
	"ratelimit limit":                           "ratelimit limit",
	"ratelimit multi-limit":                     "ratelimit multi-limit",
	"ratelimit set-override":                    "ratelimit overrides set",
	"ratelimit get-override":                    "ratelimit overrides get",
	"ratelimit delete-override":                 "ratelimit overrides delete",
	"ratelimit list-overrides":                  "ratelimit overrides list",
}
