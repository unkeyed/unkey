"use client";

import { IconPlusOutline18 } from "@unkey/icons";
import { match } from "@unkey/match";
import {
  Button,
  SettingsGroup,
  SettingsGroupTitle,
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@unkey/ui";
import type { AddRegionState } from "./cards";
import { RegionCard } from "./region-card";
import { RegionViz } from "./region-viz";
import { useCompute } from "./use-compute";

export function ComputeSettings() {
  const { draft, limits, list, hovered, actions } = useCompute();
  return (
    <SettingsGroup>
      <SettingsGroupTitle>Regions</SettingsGroupTitle>
      <div className="overflow-clip rounded-lg border bg-raised">
        <div className="divide-y divide-grayA-4">
          <RegionViz
            draft={draft}
            unavailable={new Set(list.unavailable)}
            hovered={hovered}
            onHover={actions.hover}
          />
          <div className="flex flex-col gap-3 px-4 pt-4 pb-5">
            {list.cards.map((card) => (
              <RegionCard key={card.key} card={card} limits={limits} actions={actions} />
            ))}
            <AddRegion state={list.addRegion} onAdd={actions.startAdd} />
          </div>
        </div>
      </div>
    </SettingsGroup>
  );
}

function AddRegion({ state, onAdd }: { state: AddRegionState; onAdd: () => void }) {
  return match(state)
    .with({ type: "adding" }, () => null)
    .with({ type: "available" }, () => <AddRegionButton disabled={false} onClick={onAdd} />)
    .with({ type: "blocked" }, ({ reason }) => (
      <TooltipProvider>
        <Tooltip>
          <TooltipTrigger render={<span className="inline-flex w-fit" />}>
            <AddRegionButton disabled onClick={onAdd} />
          </TooltipTrigger>
          <TooltipContent>{reason}</TooltipContent>
        </Tooltip>
      </TooltipProvider>
    ))
    .exhaustive();
}

function AddRegionButton({ disabled, onClick }: { disabled: boolean; onClick: () => void }) {
  return (
    <Button variant="outline" size="sm" className="w-fit" disabled={disabled} onClick={onClick}>
      <IconPlusOutline18 />
      Add region
    </Button>
  );
}
