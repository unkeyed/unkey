package deployment

// Change identifies a deployment transition, including distinct causes of failure.
type Change string

const (
	Preparing                 Change = "preparing"
	BuildFailed               Change = "build_failed"
	LoadFailed                Change = "load_failed"
	Cancelled                 Change = "cancelled"
	Superseded                Change = "superseded"
	RegionsStarted            Change = "regions_started"
	RegionsFailed             Change = "regions_failed"
	RoutingStarted            Change = "routing_started"
	RoutingFailed             Change = "routing_failed"
	FinalizationFailed        Change = "finalization_failed"
	Live                      Change = "live"
	CompletedWithoutPromotion Change = "completed_without_promotion"
)

// Revision orders reporting stages; alternative outcomes can share a revision.
// Unknown changes return zero.
func (c Change) Revision() int {
	switch c {
	case Preparing:
		return 1
	case BuildFailed:
		return 2
	case LoadFailed:
		return 3
	case Cancelled, Superseded:
		return 11
	case RegionsStarted:
		return 5
	case RegionsFailed:
		return 6
	case RoutingStarted:
		return 7
	case RoutingFailed:
		return 8
	case FinalizationFailed:
		return 9
	case Live, CompletedWithoutPromotion:
		return 10
	}
	return 0
}

// Terminal reports whether a known change ends deployment processing.
// Unknown changes return false.
func (c Change) Terminal() bool {
	switch c {
	case BuildFailed, LoadFailed, Cancelled, Superseded, RegionsFailed, RoutingFailed, FinalizationFailed, Live, CompletedWithoutPromotion:
		return true
	case Preparing, RegionsStarted, RoutingStarted:
		return false
	}
	return false
}

// Type maps a change to its subscription event type; multiple changes may share a type.
// Unknown changes return an empty string.
func (c Change) Type() string {
	switch c {
	case Preparing:
		return "deployment.preparing"
	case RegionsStarted:
		return "deployment.started"
	case RoutingStarted:
		return "deployment.routing"
	case BuildFailed, LoadFailed, RegionsFailed, RoutingFailed, FinalizationFailed:
		return "deployment.failed"
	case Cancelled:
		return "deployment.cancelled"
	case Superseded:
		return "deployment.superseded"
	case Live, CompletedWithoutPromotion:
		return "deployment.succeeded"
	}
	return ""
}

// Description returns notification copy, or an empty string for an unknown change.
func (c Change) Description() string {
	switch c {
	case Preparing:
		return "Preparing deployment"
	case BuildFailed:
		return "Build or image preparation failed"
	case LoadFailed:
		return "Could not load deployment after build"
	case Cancelled:
		return "Deployment stopped: cancelled"
	case Superseded:
		return "Deployment stopped: superseded"
	case RegionsStarted:
		return "Deploying to regions"
	case RegionsFailed:
		return "Deployment to regions failed"
	case RoutingStarted:
		return "Configuring routing"
	case RoutingFailed:
		return "Routing configuration failed"
	case FinalizationFailed:
		return "Finalization failed"
	case Live:
		return "Deployment is live"
	case CompletedWithoutPromotion:
		return "Deployment completed without changing live traffic"
	}
	return ""
}
