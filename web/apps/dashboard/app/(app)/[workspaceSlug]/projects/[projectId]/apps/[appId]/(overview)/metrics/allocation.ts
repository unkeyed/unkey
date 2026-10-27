import type { MetricId, ReplicaTarget } from "@/lib/metrics/app-metrics";
import type { MetricUnit } from "@/lib/metrics/views";
import { formatCpu, formatMemory, formatStorage } from "@/lib/utils/deployment-formatters";

export type LimitedMetric = Extract<MetricId, "cpu" | "memory" | "disk">;

type Allocation = {
  amount: (replica: ReplicaTarget) => number;
  label: (replica: ReplicaTarget) => string;
};

export const ALLOCATION: Record<LimitedMetric, Allocation> = {
  cpu: {
    amount: (replica) => replica.cpuMillicores,
    label: (replica) => formatCpu(replica.cpuMillicores),
  },
  memory: {
    amount: (replica) => replica.memoryMib * 2 ** 20,
    label: (replica) => formatMemory(replica.memoryMib),
  },
  disk: {
    amount: (replica) => replica.storageMib * 2 ** 20,
    label: (replica) => formatStorage(replica.storageMib),
  },
};

export type ChartScale = { unit: MetricUnit; yMax: number | null; tickCount: number };

export const SHARE_SCALE: ChartScale = { unit: "ratio", yMax: 1, tickCount: 3 };

export function absoluteScale(unit: MetricUnit): ChartScale {
  return { unit, yMax: null, tickCount: 4 };
}

export function shareOf(value: number | null, allocation: number): number | null {
  return value === null || allocation <= 0 ? null : value / allocation;
}
