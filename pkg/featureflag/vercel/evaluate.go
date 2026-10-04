package vercel

import (
	"context"
	"errors"
	"time"

	"github.com/open-feature/go-sdk/openfeature"
)

var errInvalidContext = errors.New("evaluation context must contain nested string attributes")

// BooleanEvaluation resolves flag from the current snapshot. Missing, stale,
// unsupported, or unmatched input returns defaultValue with a resolution
// error, never a guessed value.
func (p *Provider) BooleanEvaluation(_ context.Context, flag string, defaultValue bool, flatCtx openfeature.FlattenedContext) openfeature.BoolResolutionDetail {
	s := p.snapshot.Load()
	if s == nil {
		return boolError(defaultValue, openfeature.NewProviderNotReadyResolutionError("feature flag data is not ready"))
	}
	if time.Since(s.fetchedAt) > p.config.MaxStaleness {
		return boolError(defaultValue, openfeature.NewProviderNotReadyResolutionError("feature flag data is stale"))
	}

	definition, ok := s.flags[flag]
	if !ok {
		return boolError(defaultValue, openfeature.NewFlagNotFoundResolutionError("feature flag was not found"))
	}
	if definition.err != nil {
		return boolError(defaultValue, openfeature.NewParseErrorResolutionError(definition.err.Error()))
	}
	if definition.paused {
		return boolValue(definition.outcome, openfeature.StaticReason)
	}

	value, matched, err := definition.match(flatCtx)
	if err != nil {
		return boolError(defaultValue, openfeature.NewInvalidContextResolutionError(err.Error()))
	}
	if !matched {
		return boolValue(definition.outcome, openfeature.DefaultReason)
	}
	return boolValue(value, openfeature.TargetingMatchReason)
}

func (f compiledFlag) match(ctx openfeature.FlattenedContext) (bool, bool, error) {
	attributes, err := f.attributes(ctx)
	if err != nil {
		return false, false, err
	}
	for _, t := range f.targets {
		if t.matches(attributes) {
			return t.value, true, nil
		}
	}
	return false, false, nil
}

func (f compiledFlag) attributes(ctx openfeature.FlattenedContext) (map[attributePath]string, error) {
	attributes := map[attributePath]string{}
	for _, t := range f.targets {
		for path := range t.attributes {
			rawEntity, ok := ctx[path.entity]
			if !ok {
				continue
			}
			entity, ok := rawEntity.(map[string]any)
			if !ok {
				return nil, errInvalidContext
			}
			rawValue, ok := entity[path.attribute]
			if !ok {
				continue
			}
			value, ok := rawValue.(string)
			if !ok {
				return nil, errInvalidContext
			}
			attributes[path] = value
		}
	}
	return attributes, nil
}

func (t target) matches(attributes map[attributePath]string) bool {
	for path, accepted := range t.attributes {
		value, ok := attributes[path]
		if !ok {
			continue
		}
		if _, ok := accepted[value]; ok {
			return true
		}
	}
	return false
}

func boolValue(value bool, reason openfeature.Reason) openfeature.BoolResolutionDetail {
	var detail openfeature.BoolResolutionDetail
	detail.Value = value
	detail.Reason = reason
	return detail
}

func boolError(value bool, resolutionError openfeature.ResolutionError) openfeature.BoolResolutionDetail {
	detail := boolValue(value, openfeature.ErrorReason)
	detail.ResolutionError = resolutionError
	return detail
}

// StringEvaluation returns fallback because the provider supports boolean
// flags only.
func (p *Provider) StringEvaluation(_ context.Context, _ string, fallback string, _ openfeature.FlattenedContext) openfeature.StringResolutionDetail {
	return openfeature.StringResolutionDetail{Value: fallback, ProviderResolutionDetail: unsupportedType()}
}

// FloatEvaluation returns fallback because the provider supports boolean
// flags only.
func (p *Provider) FloatEvaluation(_ context.Context, _ string, fallback float64, _ openfeature.FlattenedContext) openfeature.FloatResolutionDetail {
	return openfeature.FloatResolutionDetail{Value: fallback, ProviderResolutionDetail: unsupportedType()}
}

// IntEvaluation returns fallback because the provider supports boolean flags
// only.
func (p *Provider) IntEvaluation(_ context.Context, _ string, fallback int64, _ openfeature.FlattenedContext) openfeature.IntResolutionDetail {
	return openfeature.IntResolutionDetail{Value: fallback, ProviderResolutionDetail: unsupportedType()}
}

// ObjectEvaluation returns fallback because the provider supports boolean
// flags only.
func (p *Provider) ObjectEvaluation(_ context.Context, _ string, fallback any, _ openfeature.FlattenedContext) openfeature.InterfaceResolutionDetail {
	return openfeature.InterfaceResolutionDetail{Value: fallback, ProviderResolutionDetail: unsupportedType()}
}

func unsupportedType() openfeature.ProviderResolutionDetail {
	var detail openfeature.ProviderResolutionDetail
	detail.Reason = openfeature.ErrorReason
	detail.ResolutionError = openfeature.NewTypeMismatchResolutionError("provider supports boolean flags only")
	return detail
}
