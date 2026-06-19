package ctrlclient

import (
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/auth/principal"
	"github.com/unkeyed/unkey/pkg/zen"
)

// Actor builds the ctrl.v1.ActorInfo that ctrl RPCs require to attribute audit
// logs, reading the principal and origin off the session so handlers don't
// assemble it by hand. Errors if the session has no principal (route wired
// without auth middleware).
func Actor(s *zen.Session) (*ctrlv1.ActorInfo, error) {
	p, err := s.GetPrincipal()
	if err != nil {
		return nil, err
	}

	return &ctrlv1.ActorInfo{
		Id:        p.Subject.ID,
		Name:      p.Subject.Name,
		Type:      subjectType(p.Subject.Type),
		RemoteIp:  s.Location(),
		UserAgent: s.UserAgent(),
		Meta:      make(map[string]string),
	}, nil
}

func subjectType(t principal.SubjectType) ctrlv1.ActorType {
	switch t {
	case principal.SubjectTypeUser:
		return ctrlv1.ActorType_ACTOR_TYPE_USER
	case principal.SubjectTypeRootKey:
		return ctrlv1.ActorType_ACTOR_TYPE_ROOT_KEY
	default:
		return ctrlv1.ActorType_ACTOR_TYPE_UNSPECIFIED
	}
}
