import { Bolt, ChartActivity, CircleCheck, Focus, Heart, Layers2 } from "@unkey/icons";
import type { DeploymentNode } from "../../../nodes";
import { MetricPill } from "../../../nodes/components/metric-pill";
import { StatusIndicator } from "../../../nodes/status/status-indicator";

type SentinelInstancesProps = {
  instances: DeploymentNode[];
};

export function SentinelInstances({ instances }: SentinelInstancesProps) {
  const sentinelInstances = instances.filter((node) => node.metadata.type === "sentinel");

  if (sentinelInstances.length === 0) {
    return null;
  }

  return (
    <div className="flex px-4 w-full mt-4 flex-col">
      <div className="flex items-center gap-3 w-full">
        <div className="text-gray-9 text-xs whitespace-nowrap">Sentinel instances</div>
        <div className="h-0.5 bg-grayA-3 rounded-sm flex-1 min-w-[115px]" />
      </div>
      <div className="flex flex-col gap-6 mt-5">
        {sentinelInstances.map((instance) => {
          if (instance.metadata.type !== "sentinel") {
            return null;
          }

          const { rps, cpu, memory, health } = instance.metadata;

          return (
            <div className="flex items-start" key={instance.id}>
              <div className="flex gap-3 items-center flex-1">
                <div className="bg-redA-3 dark:bg-redA-1 border rounded-md border-grayA-4 size-[22px] flex items-center justify-center gap-3">
                  <Layers2 className="text-red-9" iconSize="md-regular" />
                </div>
                <span className="text-gray-11 text-xs">{instance.label}</span>
              </div>
              <div className="flex-1 flex flex-col gap-2">
                <div className="flex items-center gap-2">
                  <MetricPill
                    icon={<ChartActivity iconSize="sm-medium" className="shrink-0" />}
                    value={rps ?? 0}
                    tooltip="Requests per second handled by this sentinel instance"
                  />
                  <MetricPill
                    icon={<Bolt iconSize="sm-medium" className="shrink-0" />}
                    value={`${cpu ?? 0}%`}
                    tooltip="CPU usage percentage"
                  />
                  <MetricPill
                    icon={<Focus iconSize="sm-medium" className="shrink-0" />}
                    value={`${memory ?? 0}%`}
                    tooltip="Memory usage percentage"
                  />
                </div>
                <div className="flex items-center gap-2">
                  <StatusIndicator
                    orientation="horizontal"
                    icon={<CircleCheck className="text-gray-9" iconSize="sm-regular" />}
                    healthStatus={health}
                    tooltip="Sentinel is online and serving traffic"
                  />
                  <StatusIndicator
                    orientation="horizontal"
                    icon={<Heart className="text-success-9" iconSize="sm-regular" />}
                    healthStatus={health}
                    tooltip="Sentinel health status"
                    showGlow={health !== "normal"}
                  />
                </div>
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}
