package cluster

import (
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/ptr"
)

// buildVerticalAutoscalingPolicy constructs the proto VPA policy from a
// DeploymentTopology row. The caller must check dt.VpaUpdateMode.Valid
// before calling.
func buildVerticalAutoscalingPolicy(dt db.DeploymentTopology) *ctrlv1.VerticalAutoscalingPolicy {
	vpa := &ctrlv1.VerticalAutoscalingPolicy{}

	switch dt.VpaUpdateMode.DeploymentTopologyVpaUpdateMode {
	case db.DeploymentTopologyVpaUpdateModeOff:
		vpa.UpdateMode = ctrlv1.VerticalAutoscalingPolicy_UPDATE_MODE_OFF
	case db.DeploymentTopologyVpaUpdateModeInitial:
		vpa.UpdateMode = ctrlv1.VerticalAutoscalingPolicy_UPDATE_MODE_INITIAL
	case db.DeploymentTopologyVpaUpdateModeRecreate:
		vpa.UpdateMode = ctrlv1.VerticalAutoscalingPolicy_UPDATE_MODE_RECREATE
	case db.DeploymentTopologyVpaUpdateModeInPlaceOrRecreate:
		vpa.UpdateMode = ctrlv1.VerticalAutoscalingPolicy_UPDATE_MODE_IN_PLACE_OR_RECREATE
	}

	if dt.VpaControlledResources.Valid {
		switch dt.VpaControlledResources.DeploymentTopologyVpaControlledResources {
		case db.DeploymentTopologyVpaControlledResourcesCpu:
			vpa.ControlledResources = ctrlv1.VerticalAutoscalingPolicy_CONTROLLED_RESOURCES_CPU
		case db.DeploymentTopologyVpaControlledResourcesMemory:
			vpa.ControlledResources = ctrlv1.VerticalAutoscalingPolicy_CONTROLLED_RESOURCES_MEMORY
		case db.DeploymentTopologyVpaControlledResourcesBoth:
			vpa.ControlledResources = ctrlv1.VerticalAutoscalingPolicy_CONTROLLED_RESOURCES_BOTH
		}
	}

	if dt.VpaControlledValues.Valid {
		switch dt.VpaControlledValues.DeploymentTopologyVpaControlledValues {
		case db.DeploymentTopologyVpaControlledValuesRequests:
			vpa.ControlledValues = ctrlv1.VerticalAutoscalingPolicy_CONTROLLED_VALUES_REQUESTS
		case db.DeploymentTopologyVpaControlledValuesRequestsAndLimits:
			vpa.ControlledValues = ctrlv1.VerticalAutoscalingPolicy_CONTROLLED_VALUES_REQUESTS_AND_LIMITS
		}
	}

	if dt.VpaCpuMinMillicores.Valid {
		vpa.CpuMinMillicores = ptr.P(uint32(dt.VpaCpuMinMillicores.Int32))
	}
	if dt.VpaCpuMaxMillicores.Valid {
		vpa.CpuMaxMillicores = ptr.P(uint32(dt.VpaCpuMaxMillicores.Int32))
	}
	if dt.VpaMemoryMinMib.Valid {
		vpa.MemoryMinMib = ptr.P(uint32(dt.VpaMemoryMinMib.Int32))
	}
	if dt.VpaMemoryMaxMib.Valid {
		vpa.MemoryMaxMib = ptr.P(uint32(dt.VpaMemoryMaxMib.Int32))
	}

	return vpa
}
