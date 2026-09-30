package vercel

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/open-feature/go-sdk/openfeature"
	"github.com/unkeyed/unkey/pkg/logger"
)

const (
	defaultEndpoint     = "https://flags.vercel.com/v1/datafile"
	defaultRefresh      = time.Minute
	defaultTimeout      = 10 * time.Second
	defaultMaxStaleness = 10 * time.Minute
	maxBodyBytes        = 4 << 20
	minRefresh          = time.Second
	maxRefresh          = time.Hour
)

// Config configures a Vercel datafile provider.
type Config struct {
	SDKKey          string
	RefreshInterval time.Duration
	HTTPTimeout     time.Duration
	MaxStaleness    time.Duration
}

// Diagnostics is a customer-data-free view of refresh health.
type Diagnostics struct {
	Ready             bool
	SnapshotAge       time.Duration
	LastRefreshError  string
	LastRefreshFailed time.Time
	RefreshFailures   uint64
}

type Provider struct {
	config   Config
	endpoint string
	client   *http.Client

	lifecycle sync.Mutex
	snapshot  atomic.Pointer[snapshot]
	failures  atomic.Uint64
	mu        sync.Mutex
	cancel    context.CancelFunc
	done      chan struct{}
	lastErr   string
	failedAt  time.Time
}

type snapshot struct {
	fetchedAt time.Time
	flags     map[string]compiledFlag
}

type compiledFlag struct {
	variants []bool
	targets  []map[string]map[string]map[string]struct{}
	outcome  int
	paused   bool
	err      error
}

// New validates config and returns a provider. Network access starts during OpenFeature initialization.
func New(config Config) (*Provider, error) {
	return newProvider(config, defaultEndpoint, &http.Client{})
}

func newProvider(config Config, endpoint string, client *http.Client) (*Provider, error) {
	if !strings.HasPrefix(config.SDKKey, "vf_server_") {
		return nil, errors.New("vercel feature flags SDK key must start with vf_server_")
	}
	if config.RefreshInterval == 0 {
		config.RefreshInterval = defaultRefresh
	}
	if config.RefreshInterval < minRefresh || config.RefreshInterval > maxRefresh {
		return nil, fmt.Errorf("refresh interval must be between %s and %s", minRefresh, maxRefresh)
	}
	if config.HTTPTimeout == 0 {
		config.HTTPTimeout = defaultTimeout
	}
	if config.HTTPTimeout <= 0 || config.HTTPTimeout > time.Minute {
		return nil, errors.New("HTTP timeout must be positive and at most 1m")
	}
	if config.MaxStaleness == 0 {
		config.MaxStaleness = defaultMaxStaleness
	}
	if config.MaxStaleness < config.RefreshInterval || config.MaxStaleness > 24*time.Hour {
		return nil, errors.New("maximum staleness must be between the refresh interval and 24h")
	}
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil {
		return nil, errors.New("endpoint must be an absolute HTTPS URL")
	}
	boundedClient := *client
	boundedClient.Timeout = config.HTTPTimeout
	boundedClient.CheckRedirect = rejectRedirect
	var provider Provider
	provider.config = config
	provider.endpoint = endpoint
	provider.client = &boundedClient
	return &provider, nil
}

func rejectRedirect(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }

func (p *Provider) Metadata() openfeature.Metadata { return openfeature.Metadata{Name: "vercel"} }
func (p *Provider) Hooks() []openfeature.Hook      { return nil }

// InitWithContext fetches the initial datafile and starts polling.
func (p *Provider) InitWithContext(ctx context.Context, _ openfeature.EvaluationContext) error {
	p.lifecycle.Lock()
	defer p.lifecycle.Unlock()
	p.mu.Lock()
	if p.cancel != nil {
		p.mu.Unlock()
		return errors.New("vercel feature flags provider already initialized")
	}
	p.mu.Unlock()
	if err := p.refresh(ctx); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		logger.Warn("feature flag refresh failed, polling continues", "provider", "vercel", "error", err.Error())
	}
	pollCtx, cancel := context.WithCancel(context.Background())
	p.mu.Lock()
	p.cancel = cancel
	p.done = make(chan struct{})
	done := p.done
	p.mu.Unlock()
	go p.poll(pollCtx, done)
	return nil
}

func (p *Provider) Init(ctx openfeature.EvaluationContext) error {
	return p.InitWithContext(context.Background(), ctx)
}

func (p *Provider) poll(ctx context.Context, done chan struct{}) {
	defer close(done)
	ticker := time.NewTicker(p.config.RefreshInterval)
	defer ticker.Stop()
	failing := p.Diagnostics().LastRefreshError != ""
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			refreshCtx, cancel := context.WithTimeout(ctx, p.config.HTTPTimeout)
			err := p.refresh(refreshCtx)
			cancel()
			if ctx.Err() != nil {
				return
			}
			switch {
			case err != nil:
				logger.Warn("feature flag refresh failed", "provider", "vercel", "error", err.Error())
			case failing:
				logger.Info("feature flag refresh recovered", "provider", "vercel")
			}
			failing = err != nil
		}
	}
}

// ShutdownWithContext stops polling and waits for it to finish.
func (p *Provider) ShutdownWithContext(ctx context.Context) error {
	p.lifecycle.Lock()
	defer p.lifecycle.Unlock()
	p.mu.Lock()
	cancel, done := p.cancel, p.done
	p.mu.Unlock()
	if cancel == nil {
		return nil
	}
	cancel()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (p *Provider) Shutdown() {
	p.mu.Lock()
	cancel, done := p.cancel, p.done
	p.mu.Unlock()
	if cancel != nil {
		cancel()
		<-done
	}
}

// Diagnostics returns bounded refresh state without flag or targeting data.
func (p *Provider) Diagnostics() Diagnostics {
	s := p.snapshot.Load()
	p.mu.Lock()
	d := Diagnostics{Ready: false, SnapshotAge: 0, LastRefreshError: p.lastErr, LastRefreshFailed: p.failedAt, RefreshFailures: p.failures.Load()}
	p.mu.Unlock()
	if s != nil {
		d.SnapshotAge = time.Since(s.fetchedAt)
		d.Ready = d.SnapshotAge <= p.config.MaxStaleness
	}
	return d
}

func (p *Provider) refresh(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.endpoint, nil)
	if err != nil {
		return p.recordFailure("request")
	}
	req.Header.Set("Authorization", "Bearer "+p.config.SDKKey)
	res, err := p.client.Do(req)
	if err != nil {
		return p.recordFailure("transport")
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return p.recordFailure(fmt.Sprintf("http_%d", res.StatusCode))
	}
	body, err := io.ReadAll(io.LimitReader(res.Body, maxBodyBytes+1))
	if err != nil {
		return p.recordFailure("body read")
	}
	if len(body) > maxBodyBytes {
		return p.recordFailure("body too large")
	}
	s, err := parseDatafile(body, time.Now())
	if err != nil {
		return p.recordFailure("datafile parse")
	}
	p.snapshot.Store(s)
	p.mu.Lock()
	p.lastErr = ""
	p.failedAt = time.Time{}
	p.mu.Unlock()
	return nil
}

func (p *Provider) recordFailure(cause string) error {
	p.failures.Add(1)
	p.mu.Lock()
	p.lastErr = cause
	p.failedAt = time.Now()
	p.mu.Unlock()
	return fmt.Errorf("vercel feature flags refresh failed: %s", cause)
}

func (p *Provider) BooleanEvaluation(_ context.Context, flag string, defaultValue bool, flatCtx openfeature.FlattenedContext) openfeature.BoolResolutionDetail {
	s := p.snapshot.Load()
	if s == nil {
		return boolError(defaultValue, openfeature.ProviderNotReadyCode, "feature flag data is not ready")
	}
	if time.Since(s.fetchedAt) > p.config.MaxStaleness {
		return boolError(defaultValue, openfeature.ProviderNotReadyCode, "feature flag data is stale")
	}
	definition, ok := s.flags[flag]
	if !ok {
		return boolError(defaultValue, openfeature.FlagNotFoundCode, "feature flag was not found")
	}
	if definition.err != nil {
		return boolError(defaultValue, openfeature.ParseErrorCode, definition.err.Error())
	}
	index := definition.outcome
	reason := openfeature.DefaultReason
	if !definition.paused {
		matched, invalid := matchTarget(definition.targets, flatCtx)
		if invalid {
			return boolError(defaultValue, openfeature.InvalidContextCode, "evaluation context must contain nested string attributes")
		}
		if matched >= 0 {
			index = matched
			reason = openfeature.TargetingMatchReason
		}
	} else {
		reason = openfeature.StaticReason
	}
	var detail openfeature.BoolResolutionDetail
	detail.Value = definition.variants[index]
	detail.Reason = reason
	return detail
}

func matchTarget(targets []map[string]map[string]map[string]struct{}, ctx openfeature.FlattenedContext) (int, bool) {
	matched := -1
	for index, target := range targets {
		for entity, attributes := range target {
			rawEntity, exists := ctx[entity]
			if !exists {
				continue
			}
			entityMap, ok := rawEntity.(map[string]any)
			if !ok {
				return -1, true
			}
			for attribute, accepted := range attributes {
				raw, exists := entityMap[attribute]
				if !exists {
					continue
				}
				value, ok := raw.(string)
				if !ok {
					return -1, true
				}
				if _, ok := accepted[value]; ok && matched < 0 {
					matched = index
				}
			}
		}
	}
	return matched, false
}

func boolError(value bool, code openfeature.ErrorCode, message string) openfeature.BoolResolutionDetail {
	var resolutionError openfeature.ResolutionError
	switch code {
	case openfeature.ProviderNotReadyCode:
		resolutionError = openfeature.NewProviderNotReadyResolutionError(message)
	case openfeature.FlagNotFoundCode:
		resolutionError = openfeature.NewFlagNotFoundResolutionError(message)
	case openfeature.ParseErrorCode:
		resolutionError = openfeature.NewParseErrorResolutionError(message)
	case openfeature.InvalidContextCode:
		resolutionError = openfeature.NewInvalidContextResolutionError(message)
	case openfeature.ProviderFatalCode, openfeature.TypeMismatchCode, openfeature.TargetingKeyMissingCode, openfeature.GeneralCode:
		resolutionError = openfeature.NewGeneralResolutionError(message)
	}
	var detail openfeature.BoolResolutionDetail
	detail.Value = value
	detail.Reason = openfeature.ErrorReason
	detail.ResolutionError = resolutionError
	return detail
}

func (p *Provider) StringEvaluation(_ context.Context, _ string, fallback string, _ openfeature.FlattenedContext) openfeature.StringResolutionDetail {
	return openfeature.StringResolutionDetail{Value: fallback, ProviderResolutionDetail: unsupportedType()}
}
func (p *Provider) FloatEvaluation(_ context.Context, _ string, fallback float64, _ openfeature.FlattenedContext) openfeature.FloatResolutionDetail {
	return openfeature.FloatResolutionDetail{Value: fallback, ProviderResolutionDetail: unsupportedType()}
}
func (p *Provider) IntEvaluation(_ context.Context, _ string, fallback int64, _ openfeature.FlattenedContext) openfeature.IntResolutionDetail {
	return openfeature.IntResolutionDetail{Value: fallback, ProviderResolutionDetail: unsupportedType()}
}
func (p *Provider) ObjectEvaluation(_ context.Context, _ string, fallback any, _ openfeature.FlattenedContext) openfeature.InterfaceResolutionDetail {
	return openfeature.InterfaceResolutionDetail{Value: fallback, ProviderResolutionDetail: unsupportedType()}
}

func unsupportedType() openfeature.ProviderResolutionDetail {
	var detail openfeature.ProviderResolutionDetail
	detail.Reason = openfeature.ErrorReason
	detail.ResolutionError = openfeature.NewTypeMismatchResolutionError("provider supports boolean flags only")
	return detail
}

type rawDatafile struct {
	Environment string                     `json:"environment"`
	Definitions map[string]json.RawMessage `json:"definitions"`
}

func parseDatafile(body []byte, fetchedAt time.Time) (*snapshot, error) {
	var data rawDatafile
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, errors.New("invalid JSON")
	}
	if data.Environment == "" || data.Definitions == nil {
		return nil, errors.New("missing required datafile fields")
	}
	flags := make(map[string]compiledFlag, len(data.Definitions))
	for key, raw := range data.Definitions {
		flag, err := parseFlag(raw, data.Environment)
		if err != nil {
			flag.err = err
		}
		flags[key] = flag
	}
	return &snapshot{fetchedAt: fetchedAt, flags: flags}, nil
}

func parseFlag(raw json.RawMessage, environment string) (compiledFlag, error) {
	var definition struct {
		Variants     []json.RawMessage          `json:"variants"`
		Environments map[string]json.RawMessage `json:"environments"`
		Experiment   json.RawMessage            `json:"experiment"`
		VariantIDs   []string                   `json:"variantIds"`
		Seed         json.RawMessage            `json:"seed"`
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&definition); err != nil {
		return compiledFlag{}, errors.New("invalid flag definition")
	}
	if len(definition.Variants) == 0 || definition.Environments == nil {
		return compiledFlag{}, errors.New("invalid flag definition")
	}
	variants := make([]bool, len(definition.Variants))
	for i, variant := range definition.Variants {
		switch string(bytes.TrimSpace(variant)) {
		case "true":
			variants[i] = true
		case "false":
		default:
			return compiledFlag{}, errors.New("variant is not boolean")
		}
	}
	if len(definition.Experiment) > 0 && string(definition.Experiment) != "null" {
		return compiledFlag{}, errors.New("experiments are unsupported")
	}
	rawEnvironment, ok := definition.Environments[environment]
	if !ok {
		return compiledFlag{}, errors.New("environment is missing")
	}
	if bytes.Equal(bytes.TrimSpace(rawEnvironment), []byte("null")) {
		return compiledFlag{}, errors.New("environment is null")
	}
	var paused int
	if err := json.Unmarshal(rawEnvironment, &paused); err == nil {
		if paused < 0 || paused >= len(definition.Variants) {
			return compiledFlag{}, errors.New("paused variant index is invalid")
		}
		return compiledFlag{variants: variants, targets: nil, outcome: paused, paused: true, err: nil}, nil
	}
	var active struct {
		Targets     []map[string]map[string][]*string `json:"targets"`
		Rules       []json.RawMessage                 `json:"rules"`
		Fallthrough json.RawMessage                   `json:"fallthrough"`
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(rawEnvironment, &fields); err != nil {
		return compiledFlag{}, errors.New("invalid environment configuration")
	}
	for field := range fields {
		if field != "targets" && field != "rules" && field != "fallthrough" {
			return compiledFlag{}, errors.New("unsupported environment configuration")
		}
	}
	if err := json.Unmarshal(rawEnvironment, &active); err != nil {
		return compiledFlag{}, errors.New("invalid environment configuration")
	}
	if len(active.Rules) != 0 {
		return compiledFlag{}, errors.New("rules are unsupported")
	}
	if bytes.Equal(bytes.TrimSpace(active.Fallthrough), []byte("null")) {
		return compiledFlag{}, errors.New("fallthrough is null")
	}
	var outcome int
	if err := json.Unmarshal(active.Fallthrough, &outcome); err != nil {
		return compiledFlag{}, errors.New("split or malformed fallthrough is unsupported")
	}
	if outcome < 0 || outcome >= len(definition.Variants) {
		return compiledFlag{}, errors.New("fallthrough variant index is invalid")
	}
	targets := make([]map[string]map[string]map[string]struct{}, len(active.Targets))
	if len(targets) > len(definition.Variants) {
		return compiledFlag{}, errors.New("target variant index is invalid")
	}
	for i, entities := range active.Targets {
		if entities == nil {
			return compiledFlag{}, errors.New("target is null")
		}
		targets[i] = make(map[string]map[string]map[string]struct{}, len(entities))
		for entity, attributes := range entities {
			if attributes == nil {
				return compiledFlag{}, errors.New("target attributes are null")
			}
			targets[i][entity] = make(map[string]map[string]struct{}, len(attributes))
			for attribute, values := range attributes {
				if values == nil {
					return compiledFlag{}, errors.New("target values are null")
				}
				set := make(map[string]struct{}, len(values))
				for _, value := range values {
					if value == nil {
						return compiledFlag{}, errors.New("target value is null")
					}
					set[*value] = struct{}{}
				}
				targets[i][entity][attribute] = set
			}
		}
	}
	return compiledFlag{variants: variants, targets: targets, outcome: outcome, paused: false, err: nil}, nil
}
