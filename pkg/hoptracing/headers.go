package hoptracing

const (
	HeaderTraceID      = "X-Unkey-Trace-Id"
	HeaderRoute        = "X-Unkey-Route"
	HeaderTiming       = "X-Unkey-Timing"
	HeaderHopCount     = "X-Unkey-Hop-Count"
	HeaderDeploymentID = "X-Unkey-Deployment-Id"
	HeaderErrorSource  = "X-Unkey-Error-Source"

	HeaderForwardedFor   = "X-Forwarded-For"
	HeaderForwardedHost  = "X-Forwarded-Host"
	HeaderForwardedProto = "X-Forwarded-Proto"
)
