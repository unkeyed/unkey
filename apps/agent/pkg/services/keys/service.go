package keys


import (
	"context"
	
	keysv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/keys/v1"
  )

  

  type KeyService struct {
	keysv1connect.UnimplementedKeysServiceHandler
  }
  

