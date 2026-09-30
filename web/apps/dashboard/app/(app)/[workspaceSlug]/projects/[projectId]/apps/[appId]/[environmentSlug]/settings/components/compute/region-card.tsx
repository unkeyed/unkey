"use client";

import { RectFlag } from "@/app/(app)/[workspaceSlug]/projects/_components/region/rect-flag";
import { regionInfo } from "@/app/(app)/[workspaceSlug]/projects/_components/region/region-info";
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
import { inheritedSummary, unschedulableIn } from "./model";
import { type ComputeCard, type ComputePage, useCardController } from "./use-compute";

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

type FooterNote = { tone: "error" | "warning" | "muted"; text: string };

const NOTE_TONE = {
  error: "text-error-11",
  warning: "text-warning-11",
  muted: "text-gray-11",
} satisfies Record<FooterNote["tone"], string>;

function footerNote(c: ComputeCard): FooterNote | null {
  if (!c.result.ok && c.result.reason === "taken") {
    return { tone: "error", text: "This region is already in use" };
  }
  const [first, ...rest] = c.blocked;
  if (first) {
    return {
      tone: "warning",
      text:
        rest.length === 0
          ? `${first} can't be scheduled. Remove or replace it to save.`
          : `${rest.length + 1} regions can't be scheduled. Remove or replace them to save.`,
    };
  }
  if (c.dirty && c.saveMode === "manual") {
    return { tone: "muted", text: "Changes apply on next deploy" };
  }
  return null;
}

function CardFooter({ leading, c }: { leading: React.ReactNode; c: ComputeCard }) {
  const note = footerNote(c);
  return (
    <div className="flex items-center justify-between gap-3 bg-grayA-2 px-5 py-3">
      {leading}
      {c.saveMode === "manual" || note ? (
        <div className="flex items-center gap-3">
          {note ? <span className={cn("text-xs", NOTE_TONE[note.tone])}>{note.text}</span> : null}
          {c.saveMode === "manual" ? (
            <Button
              variant="primary"
              size="sm"
              className="px-3"
              disabled={!c.dirty || c.blocked.length > 0}
              onClick={c.save}
            >
              Save changes
            </Button>
          ) : null}
        </div>
      ) : null}
    </div>
  );
}

function UnavailableChip() {
  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger
          render={
            <span className="rounded-sm border border-warning-6 bg-warning-3 px-1 text-[11px] leading-4 text-warning-11" />
          }
        >
          Unavailable
        </TooltipTrigger>
        <TooltipContent>This region is currently unavailable for scheduling</TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
}

function RegionSelect({ c }: { c: ComputeCard }) {
  const name = c.view.region;
  const current = name ? regionInfo(name) : null;
  return (
    <Select
      value={name}
      onValueChange={(next) => {
        if (typeof next === "string" && next !== name) {
          c.edit({ region: next });
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
        {c.options.map(({ name: option, canSchedule }) => {
          const region = regionInfo(option);
          const taken = option !== name && c.taken.has(option);
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
  defaultOpen,
  popIn,
  onRenamed,
}: {
  page: ComputePage;
  name: string;
  defaultOpen: boolean;
  popIn: boolean;
  onRenamed: (from: string, to: string) => void;
}) {
  const card = useCardController(page, { kind: "saved", name }, (to) => onRenamed(name, to));
  const [open, setOpen] = useState(defaultOpen);
  const [pop] = useState(popIn);
  const [confirmRemove, setConfirmRemove] = useState(false);
  const shared = page.base.regions.length > 1;
  const shown = regionInfo(card.view.region ?? name);
  const saved = regionInfo(name);
  const unavailable = unschedulableIn(page.available, [shown.name]).length > 0;
  return (
    <div
      className={cn(
        "overflow-hidden rounded-lg border transition-colors",
        pop && "animate-pop motion-reduce:animate-none",
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
            {unavailable ? <UnavailableChip /> : null}
          </span>
          <span className="truncate text-xs text-gray-11">
            {shared && open ? "Same settings in every region for now" : inheritedSummary(card.view)}
          </span>
        </span>
        <span className="flex size-7 shrink-0 items-center justify-center rounded-md text-gray-10 transition-colors group-hover:bg-grayA-3 group-hover:text-gray-12">
          <IconChevronExpandYOutline12 className="size-3" />
        </span>
      </button>
      {open ? (
        <div className="divide-y divide-grayA-4 border-t border-grayA-4">
          <Row title="Region" description="Where this copy of your app runs.">
            <RegionSelect c={card} />
          </Row>
          <CardSettings c={card} />
          <CardFooter
            c={card}
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
                page.commit((current) =>
                  current.regions.length > 1
                    ? { ...current, regions: current.regions.filter((r) => r.name !== name) }
                    : null,
                )
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
  const card = useCardController(page, { kind: "new" }, onSaved);
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
          <RegionSelect c={card} />
        </Row>
        <CardSettings c={card} />
        <CardFooter
          c={card}
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
