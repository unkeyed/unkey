"use client";

import {
  type Preset,
  type SizeUnlock,
  presetFor,
  resolveLimits,
  sizeOptions,
} from "@/app/(app)/[workspaceSlug]/projects/_components/compute/sizes";
import { PlansScreen } from "@/app/(app)/[workspaceSlug]/settings/billing/components/plans-screen";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { planName } from "@/lib/billing/plan-card-state";
import { useWorkspace } from "@/providers/workspace-provider";
import {
  IconChevronExpandYOutline12,
  IconLockOutline12,
  IconMicrochipOutline18,
  IconRamOutline18,
} from "@unkey/icons";
import { match } from "@unkey/match";
import { useState } from "react";
import { formatSize } from "./size-label";

type Size = Pick<Preset, "cpuMillicores" | "memoryMib">;

type SizeFieldProps = {
  size: Size;
  onChange: (size: Size) => void;
};

function unlockLabel(unlock: SizeUnlock): string {
  return match(unlock)
    .with({ type: "plan" }, ({ plan }) => planName(plan, undefined))
    .with({ type: "contact" }, () => "Contact us")
    .exhaustive();
}

export function SizeBadge({ size }: { size: Size }) {
  return (
    <span className="rounded border border-grayA-5 bg-grayA-3 px-1.5 py-0.5 font-mono text-xs font-medium text-gray-12">
      {formatSize(size).label}
    </span>
  );
}

export function Spec({ size }: { size: Size }) {
  const { cpu, memory } = formatSize(size);
  return (
    <span className="flex items-center gap-2.5 text-xs text-gray-11">
      <span className="flex items-center gap-1">
        <IconMicrochipOutline18 className="size-3.5 text-gray-10" />
        {cpu}
      </span>
      <span className="flex items-center gap-1">
        <IconRamOutline18 className="size-3.5 text-gray-10" />
        {memory}
      </span>
    </span>
  );
}

export function SizeField({ size, onChange }: SizeFieldProps) {
  const { limits } = useWorkspace();
  const [plansOpen, setPlansOpen] = useState(false);
  const current = presetFor(size.cpuMillicores, size.memoryMib);
  const options = sizeOptions(resolveLimits(limits ?? null));

  return (
    <>
      <DropdownMenu>
        <DropdownMenuTrigger
          render={
            <button
              type="button"
              aria-label="Instance size"
              className="flex h-9 w-full items-center justify-between gap-3 rounded-md border border-grayA-5 bg-raised pr-2.5 pl-1.5 text-left transition-colors hover:border-grayA-7 focus:outline-hidden focus-visible:ring-2 focus-visible:ring-gray-7 data-popup-open:border-grayA-8"
            />
          }
        >
          <span className="flex items-center gap-3">
            <SizeBadge size={size} />
            <Spec size={size} />
          </span>
          <IconChevronExpandYOutline12 className="size-3 text-gray-10" />
        </DropdownMenuTrigger>
        <DropdownMenuContent align="start" className="w-(--anchor-width) min-w-72 p-1">
          <DropdownMenuRadioGroup
            value={current?.id ?? ""}
            onValueChange={(next) => {
              const picked = options.find((option) => option.preset.id === next);
              if (picked?.type === "available") {
                onChange({
                  cpuMillicores: picked.preset.cpuMillicores,
                  memoryMib: picked.preset.memoryMib,
                });
              }
            }}
          >
            {options.map((option) =>
              option.type === "available" ? (
                <DropdownMenuRadioItem
                  key={option.preset.id}
                  value={option.preset.id}
                  closeOnClick
                  className="gap-3 px-2 py-1.5"
                >
                  <span className="w-10 font-mono text-xs font-medium text-gray-12">
                    {option.preset.label}
                  </span>
                  <Spec size={option.preset} />
                </DropdownMenuRadioItem>
              ) : (
                <DropdownMenuItem
                  key={option.preset.id}
                  onClick={() => setPlansOpen(true)}
                  className="gap-3 px-2 py-1.5 text-gray-11"
                >
                  <span className="w-10 font-mono text-xs font-medium text-gray-11">
                    {option.preset.label}
                  </span>
                  <Spec size={option.preset} />
                  <span className="ml-auto flex items-center gap-1 text-xs text-gray-11">
                    <IconLockOutline12 className="size-3" />
                    {unlockLabel(option.unlock)}
                  </span>
                </DropdownMenuItem>
              ),
            )}
          </DropdownMenuRadioGroup>
        </DropdownMenuContent>
      </DropdownMenu>
      <PlansScreen open={plansOpen} onOpenChange={setPlansOpen} reason="compute-plan" />
    </>
  );
}
