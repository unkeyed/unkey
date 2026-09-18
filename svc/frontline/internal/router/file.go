package router

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/netip"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
	frontlinev1 "github.com/unkeyed/unkey/gen/proto/frontline/v1"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/openapi/validation"
	"github.com/unkeyed/unkey/svc/frontline/internal/db"
	"google.golang.org/protobuf/encoding/protojson"
)

type fileService struct {
	routes map[string]RouteDecision
}

type fileRoute struct {
	Hostname    string                         `toml:"hostname"`
	Upstream    string                         `toml:"upstream"`
	Protocol    db.DeploymentsUpstreamProtocol `toml:"protocol"`
	Policies    []map[string]any               `toml:"policies"`
	OpenAPISpec string                         `toml:"openapi_spec"`
}

var _ Service = (*fileService)(nil)

// NewFile loads routes and policies once, without a database or file watcher.
// OpenAPI spec paths are relative to the TOML file. Invalid configuration
// returns no service, so callers cannot serve a partially loaded routing table.
func NewFile(path string) (*fileService, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg struct {
		Routes []fileRoute `toml:"routes"`
	}
	metadata, err := toml.Decode(string(data), &cfg)
	if err != nil {
		return nil, err
	}
	for _, key := range metadata.Undecoded() {
		// Policy maps are validated by the strict protobuf decoder below.
		if len(key) > 2 && key[0] == "routes" && key[1] == "policies" {
			continue
		}
		return nil, fmt.Errorf("unknown route configuration field: %s", key)
	}
	if len(cfg.Routes) == 0 {
		return nil, fault.New("at least one route is required")
	}
	s := &fileService{routes: make(map[string]RouteDecision, len(cfg.Routes))}
	for _, route := range cfg.Routes {
		hostname := normalizeHostname(route.Hostname)
		if !validHostname(hostname) {
			return nil, fmt.Errorf("invalid route hostname: %q", route.Hostname)
		}
		if _, exists := s.routes[hostname]; exists {
			return nil, fmt.Errorf("duplicate route hostname: %q", route.Hostname)
		}
		if err := validateUpstream(route.Upstream); err != nil {
			return nil, fmt.Errorf("route %s: %w", hostname, err)
		}
		policies, err := route.loadPolicies(filepath.Dir(path))
		if err != nil {
			return nil, err
		}
		protocol := route.Protocol
		if protocol == "" {
			protocol = db.DeploymentsUpstreamProtocolHttp1
		}
		if protocol != db.DeploymentsUpstreamProtocolHttp1 && protocol != db.DeploymentsUpstreamProtocolH2c {
			return nil, fmt.Errorf("route %s: protocol must be http1 or h2c", hostname)
		}
		s.routes[hostname] = RouteDecision{
			Destination:  DestinationLocalInstance,
			DeploymentID: "", EnvironmentID: "", WorkspaceID: "", ProjectID: "", AppID: "",
			LocalInstances: []db.FindInstancesByDeploymentIDRow{{
				ID: route.Hostname, Address: route.Upstream, Status: db.InstancesStatusRunning,
				WorkspaceID: "", ProjectID: "", AppID: "", RegionName: "", RegionPlatform: "",
			}},
			RemoteRegionAddress: "", UpstreamProtocol: protocol, Policies: policies,
		}
	}
	return s, nil
}

func (s *fileService) Route(_ context.Context, hostname string) (RouteDecision, error) {
	decision, ok := s.routes[normalizeHostname(hostname)]
	if !ok {
		return RouteDecision{}, fault.New("no file route for hostname: "+hostname,
			fault.Code(codes.Frontline.Routing.ConfigNotFound.URN()),
			fault.Public("Domain not configured"))
	}
	return decision, nil
}

func (s *fileService) ValidateHostname(ctx context.Context, hostname string) error {
	_, err := s.Route(ctx, hostname)
	return err
}

func (route fileRoute) loadPolicies(directory string) ([]*frontlinev1.Policy, error) {
	hostname := normalizeHostname(route.Hostname)
	var spec []byte
	if route.OpenAPISpec != "" {
		specPath := route.OpenAPISpec
		if !filepath.IsAbs(specPath) {
			specPath = filepath.Join(directory, specPath)
		}
		var err error
		spec, err = os.ReadFile(specPath)
		if err != nil {
			return nil, fmt.Errorf("route %s: %w", hostname, err)
		}
	}
	var policies []*frontlinev1.Policy
	policyIDs := make(map[string]bool)
	hasOpenAPI := false
	for _, raw := range route.Policies {
		encoded, err := json.Marshal(raw)
		if err != nil {
			return nil, err
		}
		policy := &frontlinev1.Policy{}
		if err := protojson.Unmarshal(encoded, policy); err != nil {
			return nil, fmt.Errorf("route %s: %w", hostname, err)
		}
		if policy.GetId() == "" || policyIDs[policy.GetId()] {
			return nil, fmt.Errorf("route %s: each policy requires a unique nonempty id", hostname)
		}
		policyIDs[policy.GetId()] = true
		if policy.Enabled == nil {
			return nil, fmt.Errorf("route %s policy %s: enabled must be explicit", hostname, policy.GetId())
		}
		if policy.GetConfig() == nil || policy.GetJwtauth() != nil {
			return nil, fmt.Errorf("route %s policy %s: missing or unsupported policy configuration", hostname, policy.GetId())
		}
		for _, match := range policy.GetMatch() {
			if match.GetExpr() == nil {
				return nil, fmt.Errorf("route %s policy %s: match expressions must not be empty", hostname, policy.GetId())
			}
		}
		if firewall := policy.GetFirewall(); firewall != nil && firewall.GetAction() != frontlinev1.Action_ACTION_DENY {
			return nil, fmt.Errorf("route %s policy %s: firewall action must be ACTION_DENY", hostname, policy.GetId())
		}
		if openAPI := policy.GetOpenapi(); openAPI != nil {
			hasOpenAPI = true
			if route.OpenAPISpec != "" {
				if len(openAPI.SpecYaml) != 0 {
					return nil, fmt.Errorf("route %s: choose either openapi_spec or inline spec_yaml", hostname)
				}
				openAPI.SpecYaml = spec
			}
			if len(openAPI.SpecYaml) == 0 {
				return nil, fmt.Errorf("route %s: OpenAPI policy requires a nonempty spec", hostname)
			}
			if _, err := validation.NewFromBytes(openAPI.SpecYaml); err != nil {
				return nil, fmt.Errorf("route %s: invalid OpenAPI spec: %w", hostname, err)
			}
		}
		policies = append(policies, policy)
	}
	if route.OpenAPISpec != "" && !hasOpenAPI {
		return nil, fmt.Errorf("route %s: openapi_spec requires an OpenAPI policy", hostname)
	}
	return policies, nil
}

func normalizeHostname(hostname string) string {
	return strings.ToLower(strings.TrimSuffix(hostname, "."))
}

func validHostname(hostname string) bool {
	if _, err := netip.ParseAddr(hostname); err == nil {
		return true
	}
	if len(hostname) == 0 || len(hostname) > 253 {
		return false
	}
	for _, label := range strings.Split(hostname, ".") {
		if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for _, c := range label {
			if c != '-' && (c < 'a' || c > 'z') && (c < '0' || c > '9') {
				return false
			}
		}
	}
	return true
}

func validateUpstream(address string) error {
	host, port, err := net.SplitHostPort(address)
	if err != nil || host == "" {
		return fault.New("upstream must be a host:port address")
	}
	portNumber, err := strconv.Atoi(port)
	if err != nil || portNumber < 1 || portNumber > 65535 {
		return fault.New("upstream port must be between 1 and 65535")
	}
	target, err := url.Parse("http://" + address)
	if err != nil || target.Host != address || target.User != nil || target.Path != "" || target.RawQuery != "" || target.Fragment != "" {
		return fault.New("upstream must not contain a scheme, credentials, path, query, or fragment")
	}
	return nil
}
