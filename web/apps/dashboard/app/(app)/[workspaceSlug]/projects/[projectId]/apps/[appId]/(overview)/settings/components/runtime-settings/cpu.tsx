"use client";

import { IconMicrochipOutline18 } from "nucleo-ui-outline-18";
import { formatCpuParts } from "@/lib/utils/deployment-formatters";
import { defineResourceSlider, ResourceSliderSetting } from "../shared/resource-slider";

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
  icon: <IconMicrochipOutline18 className="text-gray-12" />,
  title: "Max CPU",
  description: "Maximum CPU limit per instance. You are only charged for actual usage.",
  settingDescription:
    "Changes apply on next deploy. Contact support@unkey.com if you need higher limits.",
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
