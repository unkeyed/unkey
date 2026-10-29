package guidance

import (
	"context"
	"fmt"
	"strings"
	"unicode"

	unkey "github.com/unkeyed/sdks/api/go/v3"
	"github.com/unkeyed/sdks/api/go/v3/models/components"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
	"github.com/unkeyed/unkey/pkg/prompt"
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
	if b.client == nil {
		b.client, err = util.CreateClient(ctx, b.cmd)
		if err != nil {
			return resourceFallback(ctx, f, field)
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

func (b *resourceBrowser) browse(ctx context.Context, f *form, field string, query resourceQuery, load resourceLoader, searchable bool) (string, error) {
	for {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		page, err := load(ctx, b.client, query)
		if err != nil {
			return resourceFallback(ctx, f, field)
		}
		options := append([]prompt.SelectOption{}, page.choices...)
		if searchable {
			options = append(options, prompt.SelectOption{Key: "search", Label: "Search (empty clears filter)"})
		}
		if page.cursor != nil && *page.cursor != "" && (query.cursor == nil || *page.cursor != *query.cursor) {
			options = append(options, prompt.SelectOption{Key: "next", Label: "Next page"})
		}
		options = append(options, prompt.SelectOption{Key: "manual", Label: "Enter ID manually"})
		question := label(field)
		if len(page.choices) == 0 {
			question += " · No resources found"
		}
		choice, err := f.prompt.SelectOrdered(question, options)
		if err != nil {
			return "", err
		}
		switch choice {
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

func resourceFallback(ctx context.Context, f *form, field string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if _, err := fmt.Fprintln(f.out, "Couldn't load the list. Your key may lack list permission. Enter the ID instead."); err != nil {
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
	case "environment":
		return listEnvironments, []string{"project", "app"}, false
	case "keyId":
		if strings.HasPrefix(legacy, "root-keys ") {
			return listRootKeys, nil, false
		}
		return listKeys, []string{"apiId"}, false
	case "permissionId", "permission":
		return listPermissions, nil, true
	case "roleId", "role":
		return listRoles, nil, true
	case "identityId", "identity":
		return listIdentities, nil, true
	case "namespaceId", "namespace":
		return listNamespaces, nil, true
	case "overrideId":
		return listOverrides, []string{"namespace"}, false
	case "deploymentId":
		return listDeployments, []string{"project", "app", "environment"}, false
	case "domainId":
		return listDomains, []string{"project", "app", "environment"}, true
	case "policyId":
		return listPolicies, []string{"project", "app", "environment"}, false
	default:
		return nil, nil, false
	}
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
		return resourceOption(item.OverrideID, item.Identifier)
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
