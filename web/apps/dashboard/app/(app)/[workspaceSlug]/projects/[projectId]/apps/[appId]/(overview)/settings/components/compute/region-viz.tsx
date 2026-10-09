"use client";

import { FanOut } from "@/components/fan-out";
import { Logomark } from "@/components/logomark";
import { RegionFlag } from "@/components/region-flag";
import { sizeLabel } from "@/lib/compute/sizing";
import { regionInfo } from "@/lib/regions";
import { formatCpu, formatMemory, formatStorage } from "@/lib/utils/deployment-formatters";
import {
  IconHardDriveOutline18,
  IconLayers3Outline18,
  IconMicrochipOutline18,
  IconRamOutline18,
} from "@unkey/icons";
import { Badge } from "@unkey/ui";
import type { ReactNode } from "react";
import {
  type ComputeValues,
  formatReplicas,
} from "../../../../../_components/settings/compute/draft";

const NODE_W = 208;
const NODE_GAP = 24;

type VizProps = {
  values: ComputeValues;
  unavailable: ReadonlySet<string>;
};

export function RegionViz({ values, unavailable }: VizProps) {
  const n = Math.max(values.regions.length, 1);
  const specs = [
    ["Instances", IconLayers3Outline18, formatReplicas(values.replicas)],
    ["CPU", IconMicrochipOutline18, formatCpu(values.cpuMillicores)],
    ["RAM", IconRamOutline18, formatMemory(values.memoryMib)],
    ["Disk", IconHardDriveOutline18, formatStorage(values.storageMib)],
  ] as const;
  return (
    <div className="flex aspect-[350/128] w-full flex-col justify-center bg-grayA-2 px-6 py-5">
      <div
        className="mx-auto flex w-full flex-col"
        style={{ maxWidth: n * NODE_W + (n - 1) * NODE_GAP }}
      >
        {values.regions.length > 1 ? (
          <>
            <div className="mx-auto flex items-center gap-2 rounded-lg bg-raised px-3 py-2 shadow-sm ring-1 ring-grayA-5">
              <Logomark className="size-6 rounded-md bg-grayA-3" />
              <span className="flex flex-col">
                <span className="text-xs font-medium text-gray-12">Unkey edge</span>
                <span className="text-2xs text-gray-10">Routes to the nearest region</span>
              </span>
            </div>
            <FanOut targets={values.regions.length} className="h-10 w-full" />
          </>
        ) : null}
        <div
          className="grid"
          style={{ gridTemplateColumns: `repeat(${n}, minmax(0, 1fr))`, columnGap: NODE_GAP }}
        >
          {values.regions.map((name) => {
            const region = regionInfo(name);
            const blocked = unavailable.has(name);
            return (
              <div
                key={name}
                data-blocked={blocked}
                className="flex min-w-0 flex-col overflow-hidden rounded-lg bg-raised shadow-sm ring-1 ring-grayA-5 transition-shadow hover:ring-grayA-8 data-[blocked=true]:ring-warning-7"
              >
                <span className="flex min-w-0 items-center gap-1.5 px-3 py-2">
                  <RegionFlag region={name} shape="rect" size="sm" />
                  <span className="shrink-0 text-xs font-medium text-gray-12">{region.city}</span>
                  <span className="truncate font-mono text-2xs text-gray-10">{name}</span>
                  <span className="ml-auto flex shrink-0 items-center gap-1">
                    {blocked ? (
                      <Badge variant="warning" size="sm" className="text-3xs leading-3">
                        Unavailable
                      </Badge>
                    ) : null}
                    <Badge variant="code">{sizeLabel(values)}</Badge>
                  </span>
                </span>
                <span className="grid grid-cols-2 gap-x-3 gap-y-1 whitespace-nowrap border-t border-grayA-4 px-3 py-2 font-mono text-3xs">
                  {specs.map(([label, Icon, value]) => (
                    <Spec key={label} icon={<Icon />} label={label} value={value} />
                  ))}
                </span>
              </div>
            );
          })}
        </div>
      </div>
    </div>
  );
}

function Spec({ icon, label, value }: { icon: ReactNode; label: string; value: string }) {
  return (
    <span className="flex min-w-0 items-center gap-1">
      <span className="shrink-0 text-gray-10 [&_svg]:size-3" aria-hidden="true">
        {icon}
      </span>
      <span className="sr-only">{label}</span>
      <span className="truncate text-gray-12">{value}</span>
    </span>
  );
}
