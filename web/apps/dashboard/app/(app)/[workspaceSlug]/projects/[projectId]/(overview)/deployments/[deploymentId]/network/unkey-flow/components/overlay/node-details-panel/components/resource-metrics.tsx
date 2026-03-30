"use client";

import { trpc } from "@/lib/trpc/client";
import { Bolt, Focus, Grid } from "@unkey/icons";
import { LogsTimeseriesBarChart } from "./chart";

type MetricRowProps = {
  icon: React.ReactNode;
  label: string;
  value: React.ReactNode;
  data: Array<{ originalTimestamp: number; [key: string]: number }>;
  dataKey: string;
  color: string;
  isLoading: boolean;
};

function MetricRow({ icon, label, value, data, dataKey, color, isLoading }: MetricRowProps) {
  return (
    <div className="flex flex-col gap-3 px-4 w-full mt-5">
      <div className="flex gap-3 items-center">
        <div className="bg-grayA-3 text-gray-12 rounded-md size-[22px] items-center flex justify-center">
          {icon}
        </div>
        <span className="text-gray-11 text-xs">{label}</span>
        <div className="ml-10">{value}</div>
      </div>
      <LogsTimeseriesBarChart
        data={data}
        config={{ [dataKey]: { label, color } }}
        height={48}
        isLoading={isLoading}
        isError={false}
      />
    </div>
  );
}

function toChartData(
  points: Array<{ x: number; y: number }> | undefined,
  dataKey: string,
): Array<{ originalTimestamp: number; [key: string]: number }> {
  if (!points?.length) {
    return [];
  }
  return points.map((p) => ({ originalTimestamp: p.x, [dataKey]: p.y }));
}

type ResourceMetricsProps = {
  resourceType: "deployment" | "sentinel";
  resourceId: string;
};

export function ResourceMetrics({ resourceType, resourceId }: ResourceMetricsProps) {
  const windowHours = 1;
  const params = { resourceType, resourceId };

  const cpu = trpc.deploy.metrics.getDeploymentCpuTimeseries.useQuery(
    { ...params, windowHours },
    { refetchInterval: 15_000 },
  );

  const memory = trpc.deploy.metrics.getDeploymentMemoryTimeseries.useQuery(
    { ...params, windowHours },
    { refetchInterval: 15_000 },
  );

  const instances = trpc.deploy.metrics.getDeploymentInstanceCountTimeseries.useQuery(
    { ...params, windowHours },
    { refetchInterval: 15_000 },
  );

  const summary = trpc.deploy.metrics.getDeploymentResourceSummary.useQuery(
    params,
    { refetchInterval: 15_000 },
  );

  const cpuValue = summary.data?.avg_cpu_millicores ?? 0;
  const memValue = summary.data?.max_memory_bytes ?? 0;
  const instanceCount = summary.data?.active_instances ?? 0;
  const cpuLimit = summary.data?.avg_cpu_limit_millicores ?? 1;

  return (
    <div>
      <div className="flex px-4 w-full">
        <div className="flex items-center gap-3 w-full">
          <div className="text-gray-9 text-xs whitespace-nowrap">Runtime metrics</div>
          <div className="h-0.5 bg-grayA-3 rounded-xs flex-1 min-w-[115px]" />
        </div>
      </div>

      <MetricRow
        icon={<Grid iconSize="sm-regular" className="shrink-0" />}
        label="Active instances"
        value={
          <span className="text-gray-12 font-medium text-[13px]">
            {instanceCount}
            <span className="font-normal text-grayA-10"> vm</span>
          </span>
        }
        data={toChartData(instances.data as Array<{ x: number; y: number }>, "active_instances")}
        dataKey="active_instances"
        color="hsl(var(--error-8))"
        isLoading={instances.isLoading}
      />

      <MetricRow
        icon={<Bolt iconSize="sm-regular" className="shrink-0" />}
        label="CPU usage"
        value={
          <span className="text-gray-12 font-medium text-[13px]">
            {Math.round(cpuValue)}
            <span className="font-normal text-grayA-10"> m</span>
          </span>
        }
        data={toChartData(cpu.data as Array<{ x: number; y: number }>, "cpu_usage")}
        dataKey="cpu_usage"
        color="hsl(var(--feature-8))"
        isLoading={cpu.isLoading}
      />

      <MetricRow
        icon={<Focus iconSize="sm-regular" className="shrink-0" />}
        label="Memory usage"
        value={
          <div className="flex gap-2.5 items-center">
            <span className="text-gray-12 font-medium text-[13px]">
              {formatBytes(memValue)}
            </span>
          </div>
        }
        data={toChartData(memory.data as Array<{ x: number; y: number }>, "memory_usage")}
        dataKey="memory_usage"
        color="hsl(var(--info-8))"
        isLoading={memory.isLoading}
      />
    </div>
  );
}

function formatBytes(bytes: number): React.ReactNode {
  if (bytes === 0) {
    return (
      <>
        0<span className="font-normal text-grayA-10"> b</span>
      </>
    );
  }
  const units = ["b", "kb", "mb", "gb"];
  const i = Math.floor(Math.log(bytes) / Math.log(1024));
  const value = (bytes / 1024 ** i).toFixed(i > 1 ? 1 : 0);
  return (
    <>
      {value}
      <span className="font-normal text-grayA-10"> {units[i]}</span>
    </>
  );
}
