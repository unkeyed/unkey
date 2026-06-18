package ctrlclient

import (
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/auth/principal"
)

// Actor builds the ctrl.v1.ActorInfo that ctrl RPCs require to attribute audit
// logs to the caller. Every handler that calls a ctrl mutation passes the
// authenticated principal plus the request's origin.
func Actor(p *principal.Principal, remoteIP, userAgent string) *ctrlv1.ActorInfo {
	return &ctrlv1.ActorInfo{
		Id:        p.Subject.ID,
		Name:      p.Subject.Name,
		Type:      subjectType(p.Subject.Type),
		RemoteIp:  remoteIP,
		UserAgent: userAgent,
	}
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
