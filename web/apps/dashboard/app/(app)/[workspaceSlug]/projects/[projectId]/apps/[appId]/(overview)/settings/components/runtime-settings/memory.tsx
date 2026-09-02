"use client";

import { IconRamOutline18 } from "nucleo-ui-outline-18";
import { formatMemoryParts } from "@/lib/utils/deployment-formatters";
import { defineResourceSlider, ResourceSliderSetting } from "../shared/resource-slider";

// Memory tiers on the slider. resolveStrategy bounds these to the workspace limit
// and adds the exact limit value as a stop when it is not one of these tiers.
const MEMORY_OPTIONS = [
  { label: "256 MiB", value: 256 },
  { label: "512 MiB", value: 512 },
  { label: "1 GiB", value: 1024 },
  { label: "2 GiB", value: 2048 },
  { label: "4 GiB", value: 4096 },
  { label: "8 GiB", value: 8192 },
  { label: "16 GiB", value: 16384 },
  { label: "32 GiB", value: 32768 },
] as const;

const memoryConfig = defineResourceSlider({
  icon: <IconRamOutline18 className="text-gray-12" />,
  title: "Memory",
  description: "Memory allocation for each instance",
  settingDescription:
    "Changes apply on next deploy. Contact support@unkey.com if you need higher limits.",
  colorVar: "warningA",
  options: MEMORY_OPTIONS,
  fallback: 256,
  formatValue: formatMemoryParts,
  read: (s) => s.memoryMib,
  write: (draft, value) => {
    draft.memoryMib = value;
  },
  limitKey: "memoryMibMaxPerInstance",
});

export const Memory = () => <ResourceSliderSetting config={memoryConfig} />;
