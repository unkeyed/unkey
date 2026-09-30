"use client";

import { trpc } from "@/lib/trpc/client";
import { IconChevronExpandYOutline12 } from "@unkey/icons";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  Button,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@unkey/ui";
import { cn } from "cn";
import type React from "react";
import { useEffect, useRef, useState } from "react";
import { InstanceRange, SizeTrigger, StorageSelect } from "./controls";
import { RectFlag } from "./flag";
import { inheritedSummary, regionInfo } from "./model";
import {
  type ComputeCard,
  type ComputePage,
  type SaveMode,
  useCardController,
} from "./use-compute";

function Row({
  title,
  description,
  children,
}: {
  title: string;
  description?: string;
  children: React.ReactNode;
}) {
  return (
    <div className="flex flex-col gap-3 px-5 py-5 lg:flex-row lg:gap-6">
      <div className="flex shrink-0 flex-col gap-1 lg:w-[34%]">
        <span className="text-sm font-medium text-gray-12">{title}</span>
        {description ? <p className="text-xs leading-5 text-gray-11">{description}</p> : null}
      </div>
      <div className="min-w-0 flex-1">{children}</div>
    </div>
  );
}

function CardSettings({ c }: { c: ComputeCard }) {
  return (
    <>
      <Row title="Size" description="CPU and memory per instance.">
        <SizeTrigger c={c} />
      </Row>
      <Row
        title="Instances"
        description={`Scales between min and max on CPU. Up to ${c.limits.replicas} per region.`}
      >
        <InstanceRange c={c} />
      </Row>
      <Row title="Storage" description="Ephemeral disk per instance.">
        <StorageSelect c={c} />
      </Row>
    </>
  );
}

function CardFooter({
  leading,
  saveMode,
  dirty,
  onSave,
}: {
  leading: React.ReactNode;
  saveMode: SaveMode;
  dirty: boolean;
  onSave: () => void;
}) {
  return (
    <div className="flex items-center justify-between gap-3 bg-grayA-2 px-5 py-3">
      {leading}
      {saveMode === "manual" ? (
        <div className="flex items-center gap-3">
          {dirty ? (
            <span className="text-xs text-gray-11">Changes apply on next deploy</span>
          ) : null}
          <Button variant="primary" size="sm" className="px-3" disabled={!dirty} onClick={onSave}>
            Save changes
          </Button>
        </div>
      ) : null}
    </div>
  );
}

function RegionSelect({
  c,
  name,
  onPick,
}: {
  c: ComputeCard;
  name: string | null;
  onPick: (next: string) => void;
}) {
  const { data: available = [] } = trpc.deploy.environmentSettings.getAvailableRegions.useQuery();
  const current = name ? regionInfo(name) : null;
  return (
    <Select
      value={name}
      onValueChange={(next) => {
        if (typeof next === "string" && next !== name) {
          onPick(next);
        }
      }}
    >
      <SelectTrigger
        wrapperClassName="w-full"
        leftIcon={current ? <RectFlag flag={current.flag} size="sm" /> : undefined}
        aria-label="Region"
      >
        <SelectValue>
          {current ? (
            <span className="flex items-center gap-2">
              {current.city}
              <span className="font-mono text-xs text-gray-10">{current.name}</span>
            </span>
          ) : (
            <span className="text-gray-9">Select a region</span>
          )}
        </SelectValue>
      </SelectTrigger>
      <SelectContent>
        {available.map(({ name: option, canSchedule }) => {
          const region = regionInfo(option);
          const taken = option !== name && c.draft.regions.includes(option);
          return (
            <SelectItem key={option} value={option} disabled={!canSchedule || taken}>
              <span className="flex w-full items-center gap-2">
                <RectFlag flag={region.flag} size="sm" />
                {region.city}
                <span className="font-mono text-xs text-gray-10">{region.name}</span>
                {canSchedule ? null : (
                  <span className="ml-auto text-[11px] text-gray-10">Unavailable</span>
                )}
              </span>
            </SelectItem>
          );
        })}
      </SelectContent>
    </Select>
  );
}

export function RegionCard({
  page,
  name,
  index,
  defaultOpen,
}: {
  page: ComputePage;
  name: string;
  index: number;
  defaultOpen: boolean;
}) {
  const card = useCardController(page);
  const [open, setOpen] = useState(defaultOpen);
  const [confirmRemove, setConfirmRemove] = useState(false);
  const shared = page.base.regions.length > 1;
  const shown = regionInfo(card.draft.regions[index] ?? name);
  const saved = regionInfo(name);
  return (
    <div
      className={cn(
        "animate-pop overflow-hidden rounded-lg border transition-colors motion-reduce:animate-none",
        page.hovered === name ? "border-grayA-7" : "border-grayA-5",
      )}
    >
      <button
        type="button"
        aria-expanded={open}
        onClick={() => setOpen((o) => !o)}
        onMouseEnter={() => page.setHovered(name)}
        onMouseLeave={() => page.setHovered(null)}
        className={cn(
          "group flex w-full min-w-0 items-center gap-3 py-3 pr-3 pl-4 transition-colors focus:outline-hidden focus-visible:bg-grayA-3",
          page.hovered === name && "bg-grayA-2",
        )}
      >
        <RectFlag flag={shown.flag} />
        <span className="flex min-w-0 flex-1 flex-col gap-0.5 text-left">
          <span className="flex items-center gap-2">
            <span className="text-sm font-medium text-gray-12">{shown.city}</span>
            <span className="font-mono text-xs text-gray-10">{shown.name}</span>
          </span>
          <span className="truncate text-xs text-gray-11">
            {shared && open
              ? "Same settings in every region for now"
              : inheritedSummary(card.draft)}
          </span>
        </span>
        <span className="flex size-7 shrink-0 items-center justify-center rounded-md text-gray-10 transition-colors group-hover:bg-grayA-3 group-hover:text-gray-12">
          <IconChevronExpandYOutline12 className="size-3" />
        </span>
      </button>
      {open ? (
        <div className="divide-y divide-grayA-4 border-t border-grayA-4">
          <Row title="Region" description="Where this copy of your app runs.">
            <RegionSelect
              c={card}
              name={shown.name}
              onPick={(next) =>
                card.edit((d) => ({
                  ...d,
                  regions: d.regions.map((r) => (r === shown.name ? next : r)),
                }))
              }
            />
          </Row>
          <CardSettings c={card} />
          <CardFooter
            saveMode={card.saveMode}
            dirty={card.dirty}
            onSave={card.save}
            leading={
              shared ? (
                <Button
                  variant="outline"
                  color="danger"
                  size="sm"
                  onClick={() => setConfirmRemove(true)}
                >
                  Remove region
                </Button>
              ) : (
                <TooltipProvider>
                  <Tooltip>
                    <TooltipTrigger render={<span className="inline-flex" />}>
                      <Button variant="outline" color="danger" size="sm" disabled>
                        Remove region
                      </Button>
                    </TooltipTrigger>
                    <TooltipContent>Your app needs at least one region.</TooltipContent>
                  </Tooltip>
                </TooltipProvider>
              )
            }
          />
        </div>
      ) : null}
      <AlertDialog open={confirmRemove} onOpenChange={setConfirmRemove}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Remove {saved.city}?</AlertDialogTitle>
            <AlertDialogDescription>
              Instances in {saved.name} stop on the next deploy. Traffic moves to your other
              regions.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              color="danger"
              onClick={() =>
                page.commit({
                  ...page.base,
                  regions: page.base.regions.filter((r) => r !== name),
                })
              }
            >
              Remove region
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}

export function NewRegionCard({
  page,
  onCancel,
  onSaved,
}: {
  page: ComputePage;
  onCancel: () => void;
  onSaved: (name: string) => void;
}) {
  const card = useCardController(page);
  const picked = card.draft.regions.find((r) => !page.base.regions.includes(r)) ?? null;
  const ref = useRef<HTMLDivElement>(null);
  useEffect(() => {
    const reduce = window.matchMedia("(prefers-reduced-motion: reduce)").matches;
    ref.current?.scrollIntoView({ behavior: reduce ? "auto" : "smooth", block: "start" });
  }, []);
  return (
    <div
      ref={ref}
      className="animate-pop scroll-mt-20 overflow-hidden rounded-lg border border-dashed border-grayA-7 motion-reduce:animate-none"
    >
      <div className="flex flex-col gap-0.5 px-4 py-3">
        <span className="text-sm font-medium text-gray-12">New region</span>
        <span className="text-xs text-gray-11">
          Pick a region. It uses the same settings as your other regions for now.
        </span>
      </div>
      <div className="divide-y divide-grayA-4 border-t border-grayA-4">
        <Row title="Region" description="Where this copy of your app runs.">
          <RegionSelect
            c={card}
            name={picked}
            onPick={(next) => {
              card.edit((d) => ({
                ...d,
                regions: [...d.regions.filter((r) => r !== picked), next],
              }));
              if (card.saveMode === "autosave") {
                onSaved(next);
              }
            }}
          />
        </Row>
        <CardSettings c={card} />
        <CardFooter
          saveMode={card.saveMode}
          dirty={card.dirty && picked !== null}
          onSave={() => {
            if (!picked) {
              return;
            }
            card.save();
            onSaved(picked);
          }}
          leading={
            <Button variant="ghost" size="sm" onClick={onCancel}>
              Cancel
            </Button>
          }
        />
      </div>
    </div>
  );
}
