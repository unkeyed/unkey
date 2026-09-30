"use client";

import { IconPlusOutline18 } from "@unkey/icons";
import { Button, Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@unkey/ui";
import { useEffect, useRef, useState } from "react";
import { addRegionBlocker, unschedulableIn } from "./model";
import { NewRegionCard, RegionCard } from "./region-card";
import { RegionViz } from "./region-viz";
import { useCompute } from "./use-compute";

function useCardKeys() {
  const keys = useRef(new Map<string, number>());
  const next = useRef(0);
  return {
    keyFor: (name: string) => {
      const existing = keys.current.get(name);
      if (existing !== undefined) {
        return existing;
      }
      const key = next.current++;
      keys.current.set(name, key);
      return key;
    },
    rename: (from: string, to: string) => {
      const key = keys.current.get(from);
      if (key !== undefined && from !== to) {
        keys.current.delete(from);
        keys.current.set(to, key);
      }
    },
  };
}

function AddRegionButton({
  blocker,
  onClick,
}: {
  blocker: string | null;
  onClick: () => void;
}) {
  const button = (
    <Button
      variant="outline"
      size="sm"
      className="w-fit"
      disabled={blocker !== null}
      onClick={onClick}
    >
      <IconPlusOutline18 />
      Add region
    </Button>
  );
  if (blocker === null) {
    return button;
  }
  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger render={<span className="inline-flex w-fit" />}>{button}</TooltipTrigger>
        <TooltipContent>{blocker}</TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
}

export function ComputeSettings() {
  const page = useCompute();
  const cardKeys = useCardKeys();
  const [adding, setAdding] = useState(false);
  const [justAdded, setJustAdded] = useState<string | null>(null);
  const names = page.base.regions.map((r) => r.name);
  const addBlocker = addRegionBlocker(page.available, names);
  const addedShown = justAdded !== null && names.includes(justAdded);
  useEffect(() => {
    if (addedShown) {
      setJustAdded(null);
    }
  }, [addedShown]);
  return (
    <section className="flex flex-col gap-3">
      <h2 className="text-base font-medium text-gray-12">Regions</h2>
      <div className="overflow-clip rounded-lg border bg-raised">
        <div className="divide-y divide-grayA-4">
          <RegionViz
            draft={page.base}
            unavailable={new Set(unschedulableIn(page.available, names))}
            hovered={page.hovered}
            onHover={page.setHovered}
          />
          <div className="flex flex-col gap-3 px-4 pt-4 pb-5">
            {names.map((name, i) => (
              <RegionCard
                key={cardKeys.keyFor(name)}
                page={page}
                name={name}
                defaultOpen={name === justAdded || (i === 0 && page.saveMode === "autosave")}
                popIn={name === justAdded}
                onRenamed={cardKeys.rename}
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
              <AddRegionButton blocker={addBlocker} onClick={() => setAdding(true)} />
            )}
          </div>
        </div>
      </div>
    </section>
  );
}
