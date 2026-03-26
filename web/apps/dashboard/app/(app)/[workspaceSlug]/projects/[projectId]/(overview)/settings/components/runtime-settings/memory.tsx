"use client";

import { formatMemoryParts } from "@/lib/utils/deployment-formatters";
import { ScanCode } from "@unkey/icons";
import { type ResourceSliderConfig, ResourceSliderSetting } from "../shared/resource-slider";

const MEMORY_OPTIONS = [
  { label: "256 MiB", value: 256 },
  { label: "512 MiB", value: 512 },
  { label: "1 GiB", value: 1024 },
  { label: "2 GiB", value: 2048 },
  { label: "4 GiB", value: 4096 },
  // { label: "8 GiB", value: 8192 },
  // { label: "16 GiB", value: 16384 },
  // { label: "32 GiB", value: 32768 },
] as const;

const memoryConfig: ResourceSliderConfig = {
  icon: <ScanCode className="text-gray-12" iconSize="xl-medium" />,
  title: "Memory",
  description: "Memory allocation for each instance",
  settingDescription:
    "Changes apply on next deploy. During beta, memory is limited to 4 GiB. Please contact support@unkey.com if you need more.",
  colorVar: "warningA",
  slider: { kind: "index-mapped", options: MEMORY_OPTIONS, fallback: 256 },
  formatValue: formatMemoryParts,
  readValue: (s) => s.memoryMib,
  writeValue: (draft, value) => {
    draft.memoryMib = value;
  },
};

export const Memory = () => <ResourceSliderSetting config={memoryConfig} />;
