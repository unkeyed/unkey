package edgeredirect

import (
	"fmt"

	edgeredirectv1 "github.com/unkeyed/unkey/gen/proto/frontline/edgeredirect/v1"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"google.golang.org/protobuf/encoding/protojson"
)

// ParseConfig deserializes the bytes stored in
// frontline_routes.edge_redirect_config into the rule slice the engine
// consumes. Empty input and the legacy "{}" placeholder both decode to
// (nil, nil) so callers can fall through to no-rule behavior without a
// special case. Mirrors sentinel's engine.ParseMiddleware contract.
func ParseConfig(raw []byte) ([]*edgeredirectv1.Rule, error) {
	if len(raw) == 0 || string(raw) == "{}" {
		return nil, nil
	}

	cfg := &edgeredirectv1.Config{}
	opts := protojson.UnmarshalOptions{DiscardUnknown: true}
	if err := opts.Unmarshal(raw, cfg); err != nil {
		return nil, fault.Wrap(err,
			fault.Code(codes.Frontline.Internal.ConfigLoadFailed.URN()),
			fault.Internal(fmt.Sprintf("unable to unmarshal edge-redirect config: %s", string(raw))),
			fault.Public("The edge-redirect configuration is invalid. Please check your config or contact support at support@unkey.com."),
		)
	}

	if len(cfg.GetRules()) == 0 {
		return nil, nil
	}
	return cfg.GetRules(), nil
}
