"use client";

import { trpc } from "@/lib/trpc/client";
import { IconPlusOutline18 } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { useState } from "react";
import { NewRegionCard, RegionCard } from "./region-card";
import { RegionViz } from "./region-viz";
import { useCompute } from "./use-compute";

export function ComputeSettings() {
  const page = useCompute();
  const [adding, setAdding] = useState(false);
  const [justAdded, setJustAdded] = useState<string | null>(null);
  const { data: available } = trpc.deploy.environmentSettings.getAvailableRegions.useQuery();
  const canAdd = (available ?? []).some(
    (r) => r.canSchedule && !page.base.regions.includes(r.name),
  );
  return (
    <section className="flex flex-col gap-3">
      <h2 className="text-base font-medium text-gray-12">Regions</h2>
      <div className="overflow-clip rounded-lg border bg-raised">
        <div className="divide-y divide-grayA-4">
          <RegionViz draft={page.base} hovered={page.hovered} onHover={page.setHovered} />
          <div className="flex flex-col gap-3 px-4 pt-4 pb-5">
            {page.base.regions.map((name, i) => (
              <RegionCard
                key={name}
                page={page}
                name={name}
                index={i}
                defaultOpen={i === 0 || name === justAdded}
              />
            ))}
            {adding ? (
              <NewRegionCard
                page={page}
                onCancel={() => setAdding(false)}
                onSaved={(name) => {
                  setJustAdded(name);
                  setAdding(false);
                }}
              />
            ) : (
              <Button
                variant="outline"
                size="sm"
                className="w-fit"
                disabled={!canAdd}
                onClick={() => setAdding(true)}
              >
                <IconPlusOutline18 />
                Add region
              </Button>
            )}
          </div>
        </div>
      </div>
    </section>
  );
}
