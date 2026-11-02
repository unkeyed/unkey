package guidance

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode"

	unkey "github.com/unkeyed/sdks/api/go/v3"
	"github.com/unkeyed/sdks/api/go/v3/models/components"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
	"github.com/unkeyed/unkey/pkg/prompt"
	"golang.org/x/term"
)

func resourcePicker(_ context.Context, cmd *cli.Command, legacy string) func(context.Context, *form, string, map[string]any) (string, bool, error) {
	browser := resourceBrowser{cmd: cmd, legacy: legacy, client: nil}
	return func(ctx context.Context, f *form, field string, values map[string]any) (string, bool, error) {
		load, _, _ := resourceSource(field, legacy)
		if load == nil {
			return "", false, nil
		}
		scope := make(map[string]string)
		for name, value := range values {
			if text, ok := value.(string); ok {
				scope[name] = text
			}
		}
		value, err := browser.pick(ctx, f, field, scope)
		return value, true, err
	}
}

type resourceBrowser struct {
	cmd    *cli.Command
	legacy string
	client *unkey.Unkey
}

type resourceQuery struct {
	scope  map[string]string
	cursor *string
	search *string
}

type resourcePage struct {
	choices []prompt.SelectOption
	cursor  *string
}

type resourceLoader func(context.Context, *unkey.Unkey, resourceQuery) (resourcePage, error)

func (b *resourceBrowser) pick(ctx context.Context, f *form, field string, scope map[string]string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	load, parents, searchable := resourceSource(field, b.legacy)
	if load == nil {
		return manualResource(ctx, f, field)
	}
	if field == "deploymentId" {
		switch {
		case scope["environment"] != "":
			parents = []string{"project", "app"}
		case scope["app"] != "":
			parents = []string{"project"}
		}
	}
	if f.screen == nil {
		choice, err := f.prompt.SelectOrdered(label(field), []prompt.SelectOption{
			{Key: "browse", Label: "Browse existing resources"},
			{Key: "manual", Label: "Enter ID manually"},
		})
		if err != nil {
			return "", err
		}
		if choice == "manual" {
			return manualResource(ctx, f, field)
		}
		if choice != "browse" {
			return "", fmt.Errorf("invalid resource selection")
		}
	} else {
		for _, parent := range parents {
			if scope[parent] != "" {
				continue
			}
			choice, err := f.screen.Resource(label(field), []prompt.SelectOption{{Key: "browse", Label: "Choose " + label(parent) + " to browse"}}, false, scope[field])
			if err != nil {
				return "", err
			}
			if value, ok := strings.CutPrefix(choice, "manual:"); ok {
				return value, nil
			}
			break
		}
	}
	if b.client == nil {
		var err error
		b.client, err = util.CreateClient(ctx, b.cmd)
		if err != nil {
			return resourceFallback(ctx, f, field, err)
		}
	}
	for _, parent := range parents {
		if scope[parent] != "" {
			continue
		}
		value, err := b.pick(ctx, f, parent, scope)
		if err != nil {
			return "", err
		}
		scope[parent] = value
	}
	return b.browse(ctx, f, field, resourceQuery{scope: scope, cursor: nil, search: nil}, load, searchable)
}

func (b *resourceBrowser) browse(ctx context.Context, f *form, field string, query resourceQuery, load resourceLoader, searchable bool) (value string, err error) {
	if f.screen != nil {
		state, err := term.MakeRaw(f.screen.fd)
		if err != nil {
			return "", err
		}
		defer func() { err = errors.Join(err, term.Restore(f.screen.fd, state)) }()
	}
	for {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		var page resourcePage
		if f.screen == nil {
			page, err = load(ctx, b.client, query)
		} else {
			page, err = f.screen.loadResource(ctx, label(field), func(ctx context.Context) (resourcePage, error) {
				return load(ctx, b.client, query)
			})
		}
		if err != nil {
			if errors.Is(err, errInterrupted) || errors.Is(err, context.Canceled) {
				return "", err
			}
			return resourceFallback(ctx, f, field, err)
		}
		options := append([]prompt.SelectOption{}, page.choices...)
		if searchable && f.screen == nil {
			options = append(options, prompt.SelectOption{Key: "search", Label: "Search (empty clears filter)"})
		}
		if page.cursor != nil && *page.cursor != "" && (query.cursor == nil || *page.cursor != *query.cursor) {
			options = append(options, prompt.SelectOption{Key: "next", Label: "Next page"})
		}
		if f.screen == nil {
			options = append(options, prompt.SelectOption{Key: "manual", Label: "Enter ID manually"})
		} else if query.search != nil {
			options = append(options, prompt.SelectOption{Key: "clear", Label: "Clear search"})
		}
		question := label(field)
		if len(page.choices) == 0 {
			question += " · No resources found"
		}
		var choice string
		if f.screen == nil {
			choice, err = f.prompt.SelectOrdered(question, options)
		} else {
			choice, err = f.screen.Resource(question, options, searchable, query.scope[field])
		}
		if err != nil {
			return "", err
		}
		if value, ok := strings.CutPrefix(choice, "manual:"); ok {
			return value, nil
		}
		if text, ok := strings.CutPrefix(choice, "search:"); ok && searchable {
			query.search, query.cursor = nonempty(text), nil
			continue
		}
		switch choice {
		case "clear":
			query.search, query.cursor = nil, nil
		case "manual":
			return manualResource(ctx, f, field)
		case "search":
			if !searchable {
				return "", fmt.Errorf("search is not supported for this resource")
			}
			text, err := f.prompt.String("Search")
			if err != nil {
				return "", err
			}
			query.search, query.cursor = nonempty(strings.TrimSpace(text)), nil
		case "next":
			if page.cursor == nil || *page.cursor == "" || (query.cursor != nil && *page.cursor == *query.cursor) {
				return "", fmt.Errorf("no next page available")
			}
			query.cursor = page.cursor
		default:
			for _, option := range page.choices {
				if option.Key == choice {
					if f.selections == nil {
						f.selections = map[string]prompt.SelectOption{}
					}
					f.selections[field] = option
					return strings.TrimPrefix(choice, "id:"), nil
				}
			}
			return "", fmt.Errorf("invalid resource selection")
		}
	}
}

func manualResource(ctx context.Context, f *form, field string) (string, error) {
	for {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		value, err := f.prompt.String(label(field))
		if err != nil {
			return "", err
		}
		if value = strings.TrimSpace(value); value != "" {
			return value, nil
		}
		if _, err := fmt.Fprintln(f.out, "Enter an ID to continue."); err != nil {
			return "", err
		}
	}
}

func resourceFallback(ctx context.Context, f *form, field string, cause error) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	name := strings.TrimSuffix(label(field), " ID")
	name = strings.ToLower(name[:1]) + name[1:]
	if _, err := fmt.Fprintf(f.out, "%s\n  Enter the %s ID instead.\n", util.ErrorMessage(cause), name); err != nil {
		return "", err
	}
	return manualResource(ctx, f, field)
}

func resourceSource(field, legacy string) (resourceLoader, []string, bool) {
	switch field {
	case "project":
		return listProjects, nil, true
	case "app":
		return listApps, []string{"project"}, true
	case "appId":
		if legacy == "portal create-portal" || legacy == "portal get-portal" || legacy == "portal update-portal" {
			return listApps, []string{"project"}, true
		}
	case "environment":
		return listEnvironments, []string{"project", "app"}, false
	case "keyId":
		if strings.HasPrefix(legacy, "root-keys ") {
			return listRootKeys, nil, false
		}
		return listKeys, []string{"apiId"}, false
	case "permission":
		return listPermissions, nil, true
	case "roleId", "role":
		return listRoles, nil, true
	case "identity":
		return listIdentities, nil, true
	case "namespace":
		return listNamespaces, nil, true
	case "identifier":
		if legacy == "ratelimit get-override" || legacy == "ratelimit delete-override" {
			return listOverrides, []string{"namespace"}, false
		}
	case "deploymentId":
		return listDeployments, nil, false
	case "domain":
		if legacy == "domains get-domain" || legacy == "domains delete-domain" || legacy == "domains verify-domain" {
			return listDomains, nil, true
		}
	case "policyId":
		return listPolicies, []string{"project", "app", "environment"}, false
	}
	return nil, nil, false
}

func resourceOption(id, name string) prompt.SelectOption {
	text := id
	if name != "" {
		text = name + " (" + id + ")"
	}
	return prompt.SelectOption{Key: "id:" + id, Label: strings.Map(func(char rune) rune {
		if !unicode.IsPrint(char) {
			return ' '
		}
		return char
	}, text)}
}

func resourceResults[T any](items []T, pagination components.Pagination, option func(T) prompt.SelectOption) resourcePage {
	page := resourcePage{choices: make([]prompt.SelectOption, 0, len(items)), cursor: nil}
	for _, item := range items {
		page.choices = append(page.choices, option(item))
	}
	if pagination.HasMore {
		page.cursor = pagination.Cursor
	}
	return page
}

func nonempty(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func listProjects(ctx context.Context, client *unkey.Unkey, query resourceQuery) (resourcePage, error) {
	res, err := client.Projects.ListProjects(ctx, components.V2ProjectsListProjectsRequestBody{Limit: new(int64(20)), Cursor: query.cursor, Search: query.search})
	if err != nil {
		return resourcePage{}, err
	}
	body := res.V2ProjectsListProjectsResponseBody
	return resourceResults(body.GetData(), body.GetPagination(), func(item components.Project) prompt.SelectOption {
		return resourceOption(item.ID, item.Name)
	}), nil
}

func listApps(ctx context.Context, client *unkey.Unkey, query resourceQuery) (resourcePage, error) {
	res, err := client.Apps.ListApps(ctx, components.V2AppsListAppsRequestBody{Project: query.scope["project"], Limit: new(int64(20)), Cursor: query.cursor, Search: query.search})
	if err != nil {
		return resourcePage{}, err
	}
	body := res.V2AppsListAppsResponseBody
	return resourceResults(body.GetData(), body.GetPagination(), func(item components.App) prompt.SelectOption {
		return resourceOption(item.ID, item.Name)
	}), nil
}

func listEnvironments(ctx context.Context, client *unkey.Unkey, query resourceQuery) (resourcePage, error) {
	res, err := client.Environments.ListEnvironments(ctx, components.V2EnvironmentsListEnvironmentsRequestBody{Project: query.scope["project"], App: query.scope["app"]})
	if err != nil {
		return resourcePage{}, err
	}
	return resourceResults(res.V2EnvironmentsListEnvironmentsResponseBody.GetData(), components.Pagination{Cursor: nil, HasMore: false}, func(item components.Environment) prompt.SelectOption {
		return resourceOption(item.ID, item.Slug)
	}), nil
}

func listKeys(ctx context.Context, client *unkey.Unkey, query resourceQuery) (resourcePage, error) {
	res, err := client.Apis.ListKeys(ctx, components.V2ApisListKeysRequestBody{
		APIID: query.scope["apiId"], Limit: new(int64(20)), Cursor: query.cursor,
		ExternalID: nil, Decrypt: new(false), RevalidateKeysCache: nil,
	})
	if err != nil {
		return resourcePage{}, err
	}
	body := res.V2ApisListKeysResponseBody
	return resourceResults(body.GetData(), body.GetPagination(), func(item components.KeyResponseData) prompt.SelectOption {
		name := "Unnamed key"
		if item.Name != nil && *item.Name != "" {
			name = *item.Name
		}
		return resourceOption(item.KeyID, name)
	}), nil
}

func listRootKeys(ctx context.Context, client *unkey.Unkey, query resourceQuery) (resourcePage, error) {
	res, err := client.RootKeys.ListKeys(ctx, components.V2RootKeysListKeysRequestBody{Limit: new(int64(20)), Cursor: query.cursor})
	if err != nil {
		return resourcePage{}, err
	}
	body := res.V2RootKeysListKeysResponseBody
	return resourceResults(body.GetData(), body.GetPagination(), func(item components.V2RootKeysListKeysResponseData) prompt.SelectOption {
		name := "Unnamed root key"
		if item.Name != nil && *item.Name != "" {
			name = *item.Name
		}
		return resourceOption(item.KeyID, name)
	}), nil
}

func listPermissions(ctx context.Context, client *unkey.Unkey, query resourceQuery) (resourcePage, error) {
	res, err := client.Permissions.ListPermissions(ctx, components.V2PermissionsListPermissionsRequestBody{Limit: new(int64(20)), Cursor: query.cursor, Search: query.search})
	if err != nil {
		return resourcePage{}, err
	}
	body := res.V2PermissionsListPermissionsResponseBody
	return resourceResults(body.GetData(), body.GetPagination(), func(item components.Permission) prompt.SelectOption {
		return resourceOption(item.ID, item.Name)
	}), nil
}

func listRoles(ctx context.Context, client *unkey.Unkey, query resourceQuery) (resourcePage, error) {
	res, err := client.Permissions.ListRoles(ctx, components.V2PermissionsListRolesRequestBody{Limit: new(int64(20)), Cursor: query.cursor, Search: query.search})
	if err != nil {
		return resourcePage{}, err
	}
	body := res.V2PermissionsListRolesResponseBody
	return resourceResults(body.GetData(), body.GetPagination(), func(item components.Role) prompt.SelectOption {
		return resourceOption(item.ID, item.Name)
	}), nil
}

func listIdentities(ctx context.Context, client *unkey.Unkey, query resourceQuery) (resourcePage, error) {
	res, err := client.Identities.ListIdentities(ctx, components.V2IdentitiesListIdentitiesRequestBody{Limit: new(int64(20)), Cursor: query.cursor, Search: query.search})
	if err != nil {
		return resourcePage{}, err
	}
	body := res.V2IdentitiesListIdentitiesResponseBody
	return resourceResults(body.GetData(), body.GetPagination(), func(item components.Identity) prompt.SelectOption {
		return resourceOption(item.ID, item.ExternalID)
	}), nil
}

func listNamespaces(ctx context.Context, client *unkey.Unkey, query resourceQuery) (resourcePage, error) {
	res, err := client.Ratelimit.ListNamespaces(ctx, components.V2RatelimitListNamespacesRequestBody{Limit: new(int64(20)), Cursor: query.cursor, Search: query.search})
	if err != nil {
		return resourcePage{}, err
	}
	body := res.V2RatelimitListNamespacesResponseBody
	return resourceResults(body.GetData(), body.GetPagination(), func(item components.RatelimitNamespace) prompt.SelectOption {
		return resourceOption(item.ID, item.Name)
	}), nil
}

func listOverrides(ctx context.Context, client *unkey.Unkey, query resourceQuery) (resourcePage, error) {
	res, err := client.Ratelimit.ListOverrides(ctx, components.V2RatelimitListOverridesRequestBody{Namespace: query.scope["namespace"], Limit: new(int64(20)), Cursor: query.cursor})
	if err != nil {
		return resourcePage{}, err
	}
	body := res.V2RatelimitListOverridesResponseBody
	return resourceResults(body.GetData(), body.GetPagination(), func(item components.RatelimitOverride) prompt.SelectOption {
		return resourceOption(item.Identifier, "")
	}), nil
}

func listDeployments(ctx context.Context, client *unkey.Unkey, query resourceQuery) (resourcePage, error) {
	res, err := client.Deployments.ListDeployments(ctx, components.V2DeploymentsListDeploymentsRequestBody{
		Project: nonempty(query.scope["project"]), App: nonempty(query.scope["app"]), Environment: nonempty(query.scope["environment"]),
		Status: nil, Branch: nil, StartTime: nil, EndTime: nil, Limit: new(int64(20)), Cursor: query.cursor,
	})
	if err != nil {
		return resourcePage{}, err
	}
	body := res.V2DeploymentsListDeploymentsResponseBody
	return resourceResults(body.GetData(), body.GetPagination(), func(item components.Deployment) prompt.SelectOption {
		return resourceOption(item.ID, item.App+" / "+item.Environment+" · "+string(item.Status))
	}), nil
}

func listDomains(ctx context.Context, client *unkey.Unkey, query resourceQuery) (resourcePage, error) {
	res, err := client.Domains.ListDomains(ctx, components.V2DomainsListDomainsRequestBody{
		Project: nonempty(query.scope["project"]), App: nonempty(query.scope["app"]), Environment: nonempty(query.scope["environment"]),
		Limit: new(int64(20)), Cursor: query.cursor, Search: query.search,
	})
	if err != nil {
		return resourcePage{}, err
	}
	body := res.V2DomainsListDomainsResponseBody
	return resourceResults(body.GetData(), body.GetPagination(), func(item components.Domain) prompt.SelectOption {
		return resourceOption(item.ID, item.Domain)
	}), nil
}

func listPolicies(ctx context.Context, client *unkey.Unkey, query resourceQuery) (resourcePage, error) {
	res, err := client.Gateway.ListPolicies(ctx, components.V2GatewayListPoliciesRequestBody{
		Project: query.scope["project"], App: query.scope["app"], Environment: query.scope["environment"],
	})
	if err != nil {
		return resourcePage{}, err
	}
	return resourceResults(res.V2GatewayListPoliciesResponseBody.GetData(), components.Pagination{Cursor: nil, HasMore: false}, func(item components.PolicyResponse) prompt.SelectOption {
		return resourceOption(item.ID, item.Name)
	}), nil
}
