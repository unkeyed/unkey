// Package policyconfig owns the encode and decode rules for stored policy
// config blobs so the storage contract lives in one place. The frontline
// gateway carries its own decoder in internal/policies (ParseMiddleware) -
// keep the two in sync when the format changes.
package policyconfig

import (
	frontlinev1 "github.com/unkeyed/unkey/gen/proto/frontline/v1"
	"google.golang.org/protobuf/encoding/protojson"
)

// Marshal encodes policies into the stored blob format. protojson omits
// empty repeated fields but the dashboard's strict schema requires the
// `policies` key, so an empty config is written literally.
func Marshal(policies []*frontlinev1.Policy) ([]byte, error) {
	if len(policies) == 0 {
		return []byte(`{"policies":[]}`), nil
	}
	return protojson.Marshal(&frontlinev1.Config{Policies: policies})
}

// Parse decodes a stored policy config blob. Empty blobs and the legacy "{}"
// are valid and yield a config with no policies. Unknown fields are discarded:
// the dashboard stores a client-side `type` discriminator alongside each
// policy, and newer writers may add fields older readers don't know.
// Callers wrap the error with their own fault code and public message.
func Parse(raw []byte) (*frontlinev1.Config, error) {
	cfg := &frontlinev1.Config{}
	if len(raw) == 0 || string(raw) == "{}" {
		return cfg, nil
	}

	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(raw, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}
