"use client";

import { formatCpuParts } from "@/lib/utils/deployment-formatters";
import { ResourceSliderSetting, defineResourceSlider } from "../shared/resource-slider";

// CPU tiers on the slider. resolveStrategy bounds these to the workspace limit
// and adds the exact limit value as a stop when it is not one of these tiers.
const CPU_OPTIONS = [
  { label: "1/4 vCPU", value: 250 },
  { label: "1/2 vCPU", value: 500 },
  { label: "1 vCPU", value: 1000 },
  { label: "2 vCPU", value: 2000 },
  { label: "4 vCPU", value: 4000 },
  { label: "8 vCPU", value: 8000 },
  { label: "16 vCPU", value: 16000 },
] as const;

const cpuConfig = defineResourceSlider({
  title: "Max CPU",
  description: "CPU limit per instance, billed on actual use.",
  colorVar: "infoA",
  options: CPU_OPTIONS,
  fallback: 250,
  formatValue: formatCpuParts,
  read: (s) => s.cpuMillicores,
  write: (draft, value) => {
    draft.cpuMillicores = value;
  },
  limitKey: "cpuCoresMaxPerInstance",
  limitMultiplier: 1_000,
});

export const Cpu = () => <ResourceSliderSetting config={cpuConfig} />;
