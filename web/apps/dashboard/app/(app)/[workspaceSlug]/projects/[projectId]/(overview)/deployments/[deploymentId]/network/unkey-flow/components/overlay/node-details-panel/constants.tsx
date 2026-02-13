import type { ChartConfig } from "@/components/ui/chart";
import {
  Bolt,
  ChartActivity,
  ChevronExpandY,
  Focus,
  Grid,
  HalfDottedCirclePlay,
  Storage,
  TimeClock,
} from "@unkey/icons";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@unkey/ui";
import type { generateRealisticChartData } from "./utils";

export const metrics: Array<{
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
      <Select defaultValue="p50">
        <SelectTrigger
          className="!bg-grayA-3 rounded-full px-3 py-1.5 flex items-center gap-1.5 border-0 h-auto min-h-0! focus:border-none"
          rightIcon={
            <ChevronExpandY className="text-gray-10 absolute right-3 w-4 h-4 opacity-70" />
          }
        >
          <SelectValue />
        </SelectTrigger>
        <SelectContent className="[&_[role=option]:hover]:bg-grayA-3 [&_[role=option][data-highlighted]]:bg-grayA-2 w-56">
          <SelectItem value="p50">
            <div className="flex items-center gap-3 tabular-nums">
              <div className="bg-success-11 rounded-full size-1.5 ring-[3px] ring-successA-4 ring-offset-0" />
              <span className="text-gray-12 font-medium text-[13px]">p50</span>
              <span className="text-grayA-10 text-[13px]">3.1ms</span>
            </div>
          </SelectItem>
          <SelectItem value="p75">
            <div className="flex items-center gap-3 tabular-nums">
              <div className="bg-info-11 rounded-full size-1.5 ring-[3px] ring-infoA-4 ring-offset-0" />
              <span className="text-gray-12 font-medium text-[13px]">p75</span>
              <span className="text-grayA-10 text-[13px]">4.8ms</span>
            </div>
          </SelectItem>
          <SelectItem value="p90">
            <div className="flex items-center gap-3 tabular-nums">
              <div className="bg-feature-11 rounded-full size-1.5 ring-[3px] ring-featureA-4 ring-offset-0" />
              <span className="text-gray-12 font-medium text-[13px]">p90</span>
              <span className="text-grayA-10 text-[13px]">7.2ms</span>
            </div>
          </SelectItem>
          <SelectItem value="p95">
            <div className="flex items-center gap-3 tabular-nums">
              <div className="bg-orange-11 rounded-full size-1.5 ring-[3px] ring-orangeA-4 ring-offset-0" />
              <span className="text-gray-12 font-medium text-[13px]">p95</span>
              <span className="text-grayA-10 text-[13px]">9.5ms</span>
            </div>
          </SelectItem>
          <SelectItem value="p99">
            <div className="flex items-center gap-3 tabular-nums">
              <div className="bg-error-11 rounded-full size-1.5 ring-[3px] ring-errorA-4 ring-offset-0" />
              <span className="text-gray-12 font-medium text-[13px]">p99</span>
              <span className="text-grayA-10 text-[13px]">15.3ms</span>
            </div>
          </SelectItem>
        </SelectContent>
      </Select>
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
