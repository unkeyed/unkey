"use client";

import { EdgeNode, Spec, sizeSpecs } from "@/components/compute-diagram";
import { FanOut } from "@/components/fan-out";
import { RegionFlag } from "@/components/region-flag";
import type { EnvironmentSettings } from "@/lib/collections/deploy/environment-settings";
import { sizeLabel } from "@/lib/compute/sizing";
import { regionInfo } from "@/lib/regions";
import { IconLayers3Outline18 } from "@unkey/icons";
import { Badge } from "@unkey/ui";
import type { ReactNode } from "react";
import { formatReplicas, readCompute } from "./draft";

const NODE_W = 208;
const NODE_GAP = 24;

type VizProps = {
  settings: EnvironmentSettings;
  unavailable: ReadonlySet<string>;
  action?: ReactNode;
};

export function RegionViz({ settings, unavailable, action }: VizProps) {
  const values = readCompute(settings);
  const n = Math.max(settings.regions.length, 1);
  const size = sizeSpecs(values);
  return (
    <div className="relative flex aspect-[350/128] w-full flex-col justify-center bg-grayA-2 px-6 py-5">
      {action ? <div className="absolute top-3 right-3">{action}</div> : null}
      <div
        className="mx-auto flex w-full flex-col"
        style={{ maxWidth: n * NODE_W + (n - 1) * NODE_GAP }}
      >
        {settings.regions.length > 1 ? (
          <>
            <EdgeNode />
            <FanOut targets={settings.regions.length} className="h-10 w-full" />
          </>
        ) : null}
        <div
          className="grid"
          style={{ gridTemplateColumns: `repeat(${n}, minmax(0, 1fr))`, columnGap: NODE_GAP }}
        >
          {settings.regions.map(({ name, replicasMin, replicasMax }) => {
            const region = regionInfo(name);
            const blocked = unavailable.has(name);
            const specs = [
              {
                label: "Instances",
                Icon: IconLayers3Outline18,
                value: formatReplicas({ min: replicasMin, max: replicasMax }),
              },
              ...size,
            ];
            return (
              <div
                key={name}
                data-blocked={blocked}
                className="flex min-w-0 flex-col overflow-hidden rounded-lg bg-raised shadow-sm ring-1 ring-grayA-5 data-[blocked=true]:ring-warning-7"
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
                  {specs.map(({ label, Icon, value }) => (
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
