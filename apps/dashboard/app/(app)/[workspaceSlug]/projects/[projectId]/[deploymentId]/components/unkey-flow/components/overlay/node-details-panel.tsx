import type { ChartConfig } from "@/components/ui/chart";
import {
  Bolt,
  Book2,
  ChartActivity,
  ChevronExpandY,
  DoubleChevronRight,
  Focus,
  Grid,
  HalfDottedCirclePlay,
  Storage,
  TimeClock,
} from "@unkey/icons";
import {
  InfoTooltip,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import { CardHeader } from "../nodes/deploy-node";
import { type DeploymentNode, REGION_INFO } from "../nodes/types";
import { LogsTimeseriesBarChart } from "./chart";

type NodeDetailsPanelProps = {
  node?: DeploymentNode;
};

function generateRealisticChartData({
  count = 90,
  baseValue = 200,
  variance = 100,
  trend = 0,
  spikeProbability = 0.05,
  spikeMultiplier = 3,
  startTime = Date.now() - 90 * 24 * 60 * 60 * 1000,
  intervalMs = 24 * 60 * 60 * 1000,
  dataKey = "value",
}: {
  count?: number;
  baseValue?: number;
  variance?: number;
  trend?: number;
  spikeProbability?: number;
  spikeMultiplier?: number;
  startTime?: number;
  intervalMs?: number;
  dataKey?: string;
} = {}): Array<{
  originalTimestamp: number;
  total: number;
  [key: string]: number;
}> {
  return Array.from({ length: count }, (_, i) => {
    const trendValue = baseValue + trend * i;
    const randomVariance = (Math.random() - 0.5) * variance * 2;
    const hasSpike = Math.random() < spikeProbability;
    const spikeValue = hasSpike ? randomVariance * spikeMultiplier : 0;
    const value = Math.max(0, Math.floor(trendValue + randomVariance + spikeValue));

    return {
      originalTimestamp: startTime + i * intervalMs,
      [dataKey]: value,
      total: value, // Add total field for LogsTimeseriesBarChart
    };
  });
}

export const NodeDetailsPanel = ({ node }: NodeDetailsPanelProps) => {
  if (!node) {
    return null;
  }

  const { flagCode, zones, health } = node.metadata;
  const regionInfo = REGION_INFO[flagCode];

  const baseConfig = {
    startTime: Date.now() - 24 * 60 * 60 * 1000 * 5,
    intervalMs: 60 * 60 * 1000,
  };

  const metrics: Array<{
    icon: React.ReactNode;
    label: string;
    value: React.ReactNode;
    config: ChartConfig;
    dataConfig: Parameters<typeof generateRealisticChartData>[0];
  }> = [
    {
      icon: <Grid iconSize="sm-regular" className="shrink-0" />,
      label: "Active instances",
      value: (
        <span className="text-gray-12 font-medium text-[13px]">
          2<span className="font-normal text-grayA-10">vm</span>
        </span>
      ),
      config: {
        active_instances: {
          label: "Active instances",
          color: "hsl(var(--error-8))",
        },
      },
      dataConfig: {
        count: 80,
        baseValue: 2,
        variance: 1,
        trend: 0.02,
        spikeProbability: 0.1,
        dataKey: "active_instances",
      },
    },
    {
      icon: <ChartActivity iconSize="sm-regular" className="shrink-0" />,
      label: "Requests",
      value: (
        <span className="text-gray-12 font-medium text-[13px]">
          24<span className="font-normal text-grayA-10"> per second</span>
        </span>
      ),
      config: {
        requests: {
          label: "Requests",
          color: "hsl(var(--warning-8))",
        },
      },
      dataConfig: {
        count: 80,
        baseValue: 20,
        variance: 15,
        trend: 0.1,
        spikeProbability: 0.15,
        dataKey: "requests",
      },
    },
    {
      icon: <Bolt iconSize="sm-regular" className="shrink-0" />,
      label: "CPU usage",
      value: (
        <span className="text-gray-12 font-medium text-[13px]">
          32<span className="font-normal text-grayA-10">%</span>
        </span>
      ),
      config: {
        cpu_usage: {
          label: "CPU usage",
          color: "hsl(var(--feature-8))",
        },
      },
      dataConfig: {
        count: 80,
        baseValue: 35,
        variance: 20,
        trend: -0.05,
        spikeProbability: 0.2,
        dataKey: "cpu_usage",
      },
    },
    {
      icon: <Focus iconSize="sm-regular" className="shrink-0" />,
      label: "Memory usage",
      value: (
        <div className="flex gap-2.5 items-center">
          <span className="text-gray-12 font-medium text-[13px]">
            32<span className="font-normal text-grayA-10">%</span>
          </span>
          <span className="text-gray-12 font-medium text-[13px]">
            1.62<span className="font-normal text-grayA-10">gb</span>
          </span>
        </div>
      ),
      config: {
        memory_usage: {
          label: "Memory usage",
          color: "hsl(var(--info-8))",
        },
      },
      dataConfig: {
        count: 80,
        baseValue: 30,
        variance: 10,
        trend: 0.08,
        spikeProbability: 0.12,
        dataKey: "memory_usage",
      },
    },
    {
      icon: <Storage iconSize="sm-regular" className="shrink-0" />,
      label: "Storage usage",
      value: (
        <div className="flex gap-2.5 items-center">
          <span className="text-gray-12 font-medium text-[13px]">
            41<span className="font-normal text-grayA-10">%</span>
          </span>
          <span className="text-gray-12 font-medium text-[13px]">
            73.4<span className="font-normal text-grayA-10">mb</span>
          </span>
        </div>
      ),
      config: {
        storage_usage: {
          label: "Storage usage",
          color: "hsl(var(--cyan-8))",
        },
      },
      dataConfig: {
        count: 80,
        baseValue: 40,
        variance: 8,
        trend: 0.15,
        spikeProbability: 0.05,
        dataKey: "storage_usage",
      },
    },
    {
      icon: <TimeClock iconSize="sm-regular" className="shrink-0" />,
      label: "Latency",
      value: (
        <div className="flex gap-2.5 items-center">
          <span className="text-gray-12 font-medium text-[13px]">p50</span>
          <span className="text-gray-12 font-medium text-[13px]">3.1ms</span>
        </div>
      ),
      config: {
        latency: {
          label: "Latency",
          color: "hsl(var(--bronze-8))",
        },
      },
      dataConfig: {
        count: 80,
        baseValue: 3,
        variance: 2,
        trend: 0.01,
        spikeProbability: 0.25,
        dataKey: "latency",
      },
    },
    {
      icon: <HalfDottedCirclePlay iconSize="sm-regular" className="shrink-0" />,
      label: "Uptime",
      value: (
        <div className="flex gap-2.5 items-center">
          <span className="text-gray-12 font-medium text-[13px]">
            3<span className="font-normal text-grayA-10">h</span>
          </span>
          <span className="text-gray-12 font-medium text-[13px]">
            14<span className="font-normal text-grayA-10">m</span>
          </span>
        </div>
      ),
      config: {
        uptime: {
          label: "Uptime",
          color: "hsl(var(--success-8))",
        },
      },
      dataConfig: {
        count: 80,
        baseValue: 100,
        variance: 5,
        trend: 0,
        spikeProbability: 0.08,
        dataKey: "uptime",
      },
    },
  ];

  return (
    <div
      className={cn(
        "absolute top-14 right-4 rounded-xl bg-white dark:bg-black border border-grayA-4 shadow-[0_2px_8px_-2px_rgba(0,0,0,0.1)] pointer-events-auto min-w-[360px] max-h-[calc(100vh-80px)]",
        "transition-all duration-300 ease-out",
        node ? "opacity-100 translate-y-0" : "opacity-0 -translate-y-2 pointer-events-none",
      )}
    >
      <div className="flex flex-col items-center overflow-y-auto max-h-full pb-4">
        <div className="flex items-center justify-between h-12 border-b border-grayA-4 w-full px-3 py-2.5">
          <div className="flex gap-2.5 items-center p-2 border rounded-lg border-grayA-5 bg-grayA-2 h-[26px]">
            <Book2 className="text-gray-12" iconSize="sm-regular" />
            <span className="text-accent-12 font-medium text-[13px] leading-4">Details</span>
          </div>
          <DoubleChevronRight className="text-gray-8 shrink-0" iconSize="lg-regular" />
        </div>
        <div className="flex items-center justify-between w-full px-3 py-4">
          <CardHeader
            variant="panel"
            icon={
              <InfoTooltip
                content={`AWS region ${node.label} (${regionInfo.location})`}
                variant="primary"
                className="px-2.5 py-1 rounded-[10px] bg-white dark:bg-blackA-12 text-xs z-30"
                position={{ align: "center", side: "top", sideOffset: 5 }}
              >
                <div className="border rounded-[10px] border-grayA-3 size-12 bg-grayA-3 flex items-center justify-center">
                  <img
                    src={`/images/flags/${flagCode}.svg`}
                    alt={flagCode}
                    className="size-[22px]"
                  />
                </div>
              </InfoTooltip>
            }
            title={node.label}
            subtitle={`${zones} availability ${zones === 1 ? "zone" : "zones"}`}
            health={health}
          />
        </div>
        <div className="flex px-4 w-full">
          <div className="flex items-center gap-3 w-full">
            <div className="text-gray-9 text-xs whitespace-nowrap">Runtime metrics</div>
            <div className="h-0.5 bg-grayA-3 rounded-sm flex-1 min-w-[115px]" />
            <div className="flex items-center gap-2 shrink-0">
              <Select>
                <SelectTrigger
                  className="rounded-lg !px-2 !py-1.5 text-gray-10 text-xs !min-h-[26px]"
                  rightIcon={<ChevronExpandY className="ml-2 text-gray-10" />}
                >
                  <SelectValue placeholder="24H" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="24h">24H</SelectItem>
                  <SelectItem value="7d">7D</SelectItem>
                  <SelectItem value="30d">30D</SelectItem>
                </SelectContent>
              </Select>
              <Select>
                <SelectTrigger
                  className="rounded-lg !px-2 !py-1.5 text-gray-10 text-xs !min-h-[26px]"
                  rightIcon={<ChevronExpandY className="ml-2 text-gray-10" />}
                >
                  <SelectValue placeholder="PST" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="pst">PST</SelectItem>
                  <SelectItem value="est">EST</SelectItem>
                  <SelectItem value="utc">UTC</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>
        </div>
        {metrics.map((metric, index) => (
          // biome-ignore lint/suspicious/noArrayIndexKey: <explanation>
          <div key={index} className="flex flex-col gap-3 px-4 w-full mt-5">
            <div className="flex gap-3 items-center">
              <div className="bg-grayA-3 text-gray-12 rounded-md size-[22px] items-center flex justify-center">
                {metric.icon}
              </div>
              <span className="text-gray-11 text-xs">{metric.label}</span>
              <div className="ml-10">{metric.value}</div>
            </div>
            <LogsTimeseriesBarChart
              data={generateRealisticChartData({
                ...metric.dataConfig,
                ...baseConfig,
              })}
              config={metric.config}
              height={48}
              isLoading={false}
              isError={false}
            />
          </div>
        ))}
      </div>
    </div>
  );
};
