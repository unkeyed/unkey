"use client";

import { formatCpuParts } from "@/lib/utils/deployment-formatters";
import { Bolt } from "@unkey/icons";
import {
  type ResourceSliderConfig,
  ResourceSliderSetting,
} from "../shared/resource-slider-setting";

const CPU_OPTIONS = [
  { label: "1/4 vCPU", value: 256 },
  { label: "1/2 vCPU", value: 512 },
  { label: "1 vCPU", value: 1024 },
  { label: "2 vCPU", value: 2048 },
  //{ label: "4 vCPU", value: 4096 },
  //{ label: "8 vCPU", value: 8192 },
  //{ label: "16 vCPU", value: 16384 },
  //{ label: "32 vCPU", value: 32768 },
] as const;

const cpuConfig: ResourceSliderConfig = {
  icon: <Bolt className="text-gray-12" iconSize="xl-medium" />,
  title: "CPU",
  description: "CPU allocation for each instance",
  settingDescription:
    "Changes apply on next deploy. During beta, CPU is limited to 2 vCPUs. Please contact support@unkey.com if you need more.",
  colorVar: "infoA",
  slider: { kind: "index-mapped", options: CPU_OPTIONS, fallback: 256 },
  formatValue: formatCpuParts,
  readValue: (s) => s.cpuMillicores,
  writeValue: (draft, value) => {
    draft.cpuMillicores = value;
  },
};

export const Cpu = () => <ResourceSliderSetting config={cpuConfig} />;
