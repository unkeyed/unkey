"use client";

import { RectFlag } from "@/app/(app)/[workspaceSlug]/projects/_components/region/rect-flag";
import { regionInfo } from "@/app/(app)/[workspaceSlug]/projects/_components/region/region-info";
import { trpc } from "@/lib/trpc/client";
import { IconChevronExpandYOutline12 } from "@unkey/icons";
import { Checkbox, Popover, PopoverContent, PopoverTrigger, Skeleton } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import { useId } from "react";

type RegionSelectProps = {
  regions: string[];
  error?: string;
  onChange: (regions: string[]) => void;
};

export function toggleRegion(selected: readonly string[], region: string): string[] {
  if (!selected.includes(region)) {
    return [...selected, region];
  }
  return selected.length > 1 ? selected.filter((name) => name !== region) : [...selected];
}

export function RegionSelect({ regions, error, onChange }: RegionSelectProps) {
  const fieldId = useId();
  const { data: availableRegions } = trpc.deploy.environmentSettings.getAvailableRegions.useQuery();
  const schedulable = availableRegions
    ?.filter((region) => region.canSchedule)
    .map((region) => regionInfo(region.name));

  if (schedulable === undefined) {
    return <Skeleton className="h-9 w-full rounded-lg" />;
  }

  const picked = schedulable.filter((region) => regions.includes(region.name));
  const first = picked.at(0);

  return (
    <div className="flex flex-col gap-1.5">
      <Popover>
        <PopoverTrigger
          aria-label="Regions"
          className={cn(
            "flex h-9 w-full items-center gap-2 rounded-lg border border-input bg-raised px-3 text-left text-sm text-grayA-12 transition-colors duration-300 hover:border-strong focus:border-gray-12 focus:ring-3 focus:ring-gray-5 focus-visible:outline-hidden",
            error && "border-error-9",
          )}
        >
          {first ? (
            <>
              <span className="flex shrink-0 items-center -space-x-1">
                {picked.slice(0, 3).map((region) => (
                  <RectFlag key={region.name} flag={region.flag} size="sm" />
                ))}
              </span>
              <span className="min-w-0 flex-1 truncate">
                {first.city}
                {picked.length > 1 ? (
                  <span className="text-gray-10"> and {picked.length - 1} more</span>
                ) : null}
              </span>
            </>
          ) : (
            <span className="flex-1 text-grayA-8">Select regions</span>
          )}
          <IconChevronExpandYOutline12 className="size-3 shrink-0 text-gray-11" />
        </PopoverTrigger>
        <PopoverContent
          align="start"
          className="w-(--anchor-width) p-1"
          aria-label="Choose regions"
        >
          {schedulable.map((region) => {
            const selected = regions.includes(region.name);
            const onlyOne = selected && regions.length === 1;
            return (
              <label
                key={region.name}
                htmlFor={`${fieldId}-${region.name}`}
                className="flex cursor-pointer items-center gap-3 rounded-md px-2 py-2 hover:bg-grayA-3"
              >
                <RectFlag flag={region.flag} size="sm" />
                <span className="flex min-w-0 flex-1 items-center gap-2">
                  <span className="truncate text-sm text-gray-12">{region.city}</span>
                  <span className="font-mono text-xs text-gray-10">{region.name}</span>
                </span>
                <Checkbox
                  id={`${fieldId}-${region.name}`}
                  checked={selected}
                  disabled={onlyOne}
                  onCheckedChange={() => onChange(toggleRegion(regions, region.name))}
                />
              </label>
            );
          })}
        </PopoverContent>
      </Popover>
      {error ? <p className="text-sm text-error-11">{error}</p> : null}
    </div>
  );
}
