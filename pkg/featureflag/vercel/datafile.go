package vercel

import (
	"bytes"
	"encoding/json"
	"errors"
	"time"
)

type snapshot struct {
	fetchedAt time.Time
	flags     map[string]compiledFlag
}

type compiledFlag struct {
	targets []target
	outcome bool
	paused  bool
	err     error
}

type target struct {
	value      bool
	attributes map[attributePath]map[string]struct{}
}

type attributePath struct {
	entity    string
	attribute string
}

type rawDatafile struct {
	Environment string                     `json:"environment"`
	Definitions map[string]json.RawMessage `json:"definitions"`
}

type rawDefinition struct {
	Variants     []json.RawMessage          `json:"variants"`
	Environments map[string]json.RawMessage `json:"environments"`
	Experiment   json.RawMessage            `json:"experiment"`
	VariantIDs   []string                   `json:"variantIds"`
	Seed         json.RawMessage            `json:"seed"`
}

type rawActiveEnvironment struct {
	Targets     []map[string]map[string][]*string `json:"targets"`
	Rules       []json.RawMessage                 `json:"rules"`
	Fallthrough json.RawMessage                   `json:"fallthrough"`
}

var activeEnvironmentFields = map[string]struct{}{"targets": {}, "rules": {}, "fallthrough": {}}

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
		flag.err = err
		flags[key] = flag
	}
	return &snapshot{fetchedAt: fetchedAt, flags: flags}, nil
}

func parseFlag(raw json.RawMessage, environment string) (compiledFlag, error) {
	var definition rawDefinition
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&definition); err != nil {
		return compiledFlag{}, errors.New("invalid flag definition")
	}
	if len(definition.Variants) == 0 || definition.Environments == nil {
		return compiledFlag{}, errors.New("invalid flag definition")
	}

	variants, err := parseVariants(definition.Variants)
	if err != nil {
		return compiledFlag{}, err
	}
	if len(definition.Experiment) > 0 && !isNull(definition.Experiment) {
		return compiledFlag{}, errors.New("experiments are unsupported")
	}

	rawEnvironment, ok := definition.Environments[environment]
	if !ok {
		return compiledFlag{}, errors.New("environment is missing")
	}
	return parseEnvironment(variants, rawEnvironment)
}

func parseVariants(raw []json.RawMessage) ([]bool, error) {
	variants := make([]bool, len(raw))
	for i, variant := range raw {
		switch string(bytes.TrimSpace(variant)) {
		case "true":
			variants[i] = true
		case "false":
		default:
			return nil, errors.New("variant is not boolean")
		}
	}
	return variants, nil
}

// parseEnvironment reads either a paused environment, which is a bare variant
// index, or an active one with targets and a fixed fallthrough variant.
func parseEnvironment(variants []bool, raw json.RawMessage) (compiledFlag, error) {
	if isNull(raw) {
		return compiledFlag{}, errors.New("environment is null")
	}

	var paused int
	if err := json.Unmarshal(raw, &paused); err == nil {
		if !validVariant(paused, variants) {
			return compiledFlag{}, errors.New("paused variant index is invalid")
		}
		return compiledFlag{targets: nil, outcome: variants[paused], paused: true, err: nil}, nil
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return compiledFlag{}, errors.New("invalid environment configuration")
	}
	for field := range fields {
		if _, ok := activeEnvironmentFields[field]; !ok {
			return compiledFlag{}, errors.New("unsupported environment configuration")
		}
	}

	var active rawActiveEnvironment
	if err := json.Unmarshal(raw, &active); err != nil {
		return compiledFlag{}, errors.New("invalid environment configuration")
	}
	if len(active.Rules) != 0 {
		return compiledFlag{}, errors.New("rules are unsupported")
	}
	if isNull(active.Fallthrough) {
		return compiledFlag{}, errors.New("fallthrough is null")
	}
	var outcome int
	if err := json.Unmarshal(active.Fallthrough, &outcome); err != nil {
		return compiledFlag{}, errors.New("split or malformed fallthrough is unsupported")
	}
	if !validVariant(outcome, variants) {
		return compiledFlag{}, errors.New("fallthrough variant index is invalid")
	}

	targets, err := parseTargets(active.Targets, variants)
	if err != nil {
		return compiledFlag{}, err
	}
	return compiledFlag{targets: targets, outcome: variants[outcome], paused: false, err: nil}, nil
}

func parseTargets(raw []map[string]map[string][]*string, variants []bool) ([]target, error) {
	if len(raw) > len(variants) {
		return nil, errors.New("target variant index is invalid")
	}

	targets := make([]target, len(raw))
	for i, entities := range raw {
		if entities == nil {
			return nil, errors.New("target is null")
		}
		targets[i] = target{value: variants[i], attributes: map[attributePath]map[string]struct{}{}}
		for entity, attributes := range entities {
			if attributes == nil {
				return nil, errors.New("target attributes are null")
			}
			for attribute, values := range attributes {
				if values == nil {
					return nil, errors.New("target values are null")
				}
				accepted := make(map[string]struct{}, len(values))
				for _, value := range values {
					if value == nil {
						return nil, errors.New("target value is null")
					}
					accepted[*value] = struct{}{}
				}
				targets[i].attributes[attributePath{entity: entity, attribute: attribute}] = accepted
			}
		}
	}
	return targets, nil
}

func validVariant(index int, variants []bool) bool {
	return index >= 0 && index < len(variants)
}

func isNull(raw json.RawMessage) bool {
	return bytes.Equal(bytes.TrimSpace(raw), []byte("null"))
}
