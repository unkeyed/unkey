"use client";

import type { EnvironmentSettings } from "@/lib/collections/deploy/environment-settings";
import { mapRegionToFlag } from "@/lib/trpc/routers/deploy/network/utils";
import { Connections3 } from "@unkey/icons";
import { RegionFlag } from "../../../../components/region-flag";
import { type ResourceSliderConfig, ResourceSliderSetting } from "../shared/resource-slider";

const formatInstanceParts = (n: number) => ({
  value: String(n),
  unit: `instance${n !== 1 ? "s" : ""}`,
});

const RegionFlags = ({ settings }: { settings: EnvironmentSettings }) => {
  const regions = settings.regions.map((r) => r.name);
  if (regions.length === 0) {
    return null;
  }
  return (
    <div className="flex items-center gap-1.5">
      {regions.map((r) => (
        <RegionFlag
          key={r}
          flagCode={mapRegionToFlag(r)}
          size="xs"
          shape="circle"
          className="[&_img]:size-3"
        />
      ))}
    </div>
  );
};

const instancesConfig: ResourceSliderConfig = {
  icon: <Connections3 className="text-gray-12" iconSize="xl-medium" />,
  title: "Max Instances",
  description:
    "Maximum number of instances per region. Scales down automatically based on CPU usage.",
  settingDescription:
    "Changes apply on next deploy. Your deployment scales up to this or down to 1 when load is low. During beta, instances are limited to 4 per region.",
  colorVar: "featureA",
  slider: { kind: "direct", min: 1, max: 4, step: 1 },
  formatValue: formatInstanceParts,
  readValue: (s) => s.regions[0]?.replicas ?? 1,
  writeValue: (draft, value) => {
    for (const region of draft.regions) {
      region.replicas = value;
    }
  },
  extraSaveChecks: (settings) => {
    const anyHasRegions = settings.some((s) => s.regions.length > 0);
    if (!anyHasRegions) {
      return {
        status: "disabled",
        reason: "Select at least one region before setting instance count",
      };
    }
    return null;
  },
  sliderAdornment: (s) => <RegionFlags settings={s} />,
};

export const Instances = () => <ResourceSliderSetting config={instancesConfig} />;
