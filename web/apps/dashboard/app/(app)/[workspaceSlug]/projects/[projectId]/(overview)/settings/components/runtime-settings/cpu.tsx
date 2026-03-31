"use client";

import { formatCpuParts } from "@/lib/utils/deployment-formatters";
import { Bolt } from "@unkey/icons";
import { type ResourceSliderConfig, ResourceSliderSetting } from "../shared/resource-slider";

const CPU_OPTIONS = [
  { label: "1/4 vCPU", value: 250 },
  { label: "1/2 vCPU", value: 500 },
  { label: "1 vCPU", value: 1000 },
  { label: "2 vCPU", value: 2000 },
  //{ label: "4 vCPU", value: 4000 },
  //{ label: "8 vCPU", value: 8000 },
  //{ label: "16 vCPU", value: 16000 },
  //{ label: "32 vCPU", value: 32000 },
] as const;

const cpuConfig: ResourceSliderConfig = {
  icon: <Bolt className="text-gray-12" iconSize="xl-medium" />,
  title: "Max CPU",
  description: "Maximum CPU limit per instance. You are only charged for actual usage.",
  settingDescription:
    "Changes apply on next deploy. During beta, CPU is limited to 2 vCPUs. Please contact support@unkey.com if you need more.",
  colorVar: "infoA",
  slider: { kind: "index-mapped", options: CPU_OPTIONS, fallback: 250 },
  formatValue: formatCpuParts,
  readValue: (s) => s.cpuMillicores,
  writeValue: (draft, value) => {
    draft.cpuMillicores = value;
  },
};

export const Cpu = () => <ResourceSliderSetting config={cpuConfig} />;
