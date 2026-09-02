"use client";

import { IconDatabaseOutline18 } from "nucleo-ui-outline-18";
import { formatStorageParts } from "@/lib/utils/deployment-formatters";
import { defineResourceSlider, ResourceSliderSetting } from "../shared/resource-slider";

// Storage tiers on the slider. resolveStrategy bounds these to the workspace
// limit and adds the exact limit value as a stop when it is not one of these tiers.
const STORAGE_OPTIONS = [
  { label: "None", value: 0 },
  { label: "512 MiB", value: 512 },
  { label: "1 GiB", value: 1024 },
  { label: "2 GiB", value: 2048 },
  { label: "5 GiB", value: 5120 },
  { label: "10 GiB", value: 10240 },
  { label: "20 GiB", value: 20480 },
  { label: "50 GiB", value: 51200 },
] as const;

const storageConfig = defineResourceSlider({
  icon: <IconDatabaseOutline18 className="text-gray-12" />,
  title: "Storage",
  description: "Ephemeral disk space per instance",
  settingDescription:
    "We wipe this volume when the instance stops, so don't keep anything you need on it. Changes apply on next deploy. Contact support@unkey.com for larger volumes.",
  colorVar: "successA",
  options: STORAGE_OPTIONS,
  fallback: 0,
  formatValue: formatStorageParts,
  read: (s) => s.storageMib,
  write: (draft, value) => {
    draft.storageMib = value;
  },
  limitKey: "storageMibMaxPerInstance",
});

export const Storage = () => <ResourceSliderSetting config={storageConfig} />;
