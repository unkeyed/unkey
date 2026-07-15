"use client";

import { formatStorageParts } from "@/lib/utils/deployment-formatters";
import { Database } from "@unkey/icons";
import { type ResourceSliderConfig, ResourceSliderSetting } from "../shared/resource-slider";

const STORAGE_OPTIONS = [
  { label: "None", value: 0 },
  { label: "512 MiB", value: 512 },
  { label: "1 GiB", value: 1024 },
  { label: "2 GiB", value: 2048 },
  { label: "5 GiB", value: 5120 },
  { label: "10 GiB", value: 10240 },
] as const;

const storageConfig: ResourceSliderConfig = {
  icon: <Database className="text-gray-12" iconSize="xl-medium" />,
  title: "Storage",
  description: "Ephemeral disk space per instance",
  settingDescription:
    "Dedicated EBS volume destroyed when the instance stops. Changes apply on next deploy. During beta, storage is limited to 10 GiB. Contact support@unkey.com for larger volumes.",
  colorVar: "successA",
  slider: { kind: "index-mapped", options: STORAGE_OPTIONS, fallback: 0 },
  formatValue: formatStorageParts,
  readValue: (s) => s.storageMib,
  writeValue: (draft, value) => {
    draft.storageMib = value;
  },
};

export const Storage = () => <ResourceSliderSetting config={storageConfig} />;
