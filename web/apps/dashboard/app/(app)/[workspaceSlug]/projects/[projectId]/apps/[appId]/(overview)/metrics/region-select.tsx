"use client";

import { RegionFlag } from "@/components/region-flag";
import { regionInfo } from "@/lib/regions";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@unkey/ui";

const ALL_REGIONS = "all";

export function RegionSelect({
  regions,
  value,
  onChange,
}: {
  regions: string[];
  value: string | null;
  onChange: (region: string | null) => void;
}) {
  const labels: Record<string, string> = {
    [ALL_REGIONS]: "All regions",
    ...Object.fromEntries(regions.map((name) => [name, regionInfo(name).city])),
  };
  return (
    <Select
      value={value ?? ALL_REGIONS}
      onValueChange={(next) => onChange(!next || next === ALL_REGIONS ? null : next)}
      items={labels}
    >
      <SelectTrigger size="sm" wrapperClassName="w-fit" aria-label="Region">
        <SelectValue />
      </SelectTrigger>
      <SelectContent>
        <SelectItem value={ALL_REGIONS}>{labels[ALL_REGIONS]}</SelectItem>
        {regions.map((name) => (
          <SelectItem key={name} value={name}>
            <span className="flex items-center gap-1.5">
              <RegionFlag region={name} shape="rect" size="xs" />
              {labels[name]}
            </span>
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}
