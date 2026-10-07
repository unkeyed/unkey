"use client";

import { RectFlag } from "@/components/region-flag";
import type { ComputeLimits } from "@/lib/compute/sizing";
import { regionInfo } from "@/lib/regions";
import { IconChevronExpandYOutline12 } from "@unkey/icons";
import { match } from "@unkey/match";
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
  SettingsRow,
  SettingsRowContent,
  SettingsRowDescription,
  SettingsRowHeader,
  SettingsRowTitle,
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@unkey/ui";
import { cn } from "cn";
import type React from "react";
import { useState } from "react";
import type { FooterLeading, FooterNote, RegionCardState, SaveButton } from "./cards";
import { InstanceRange, SizeTrigger, StorageSelect } from "./controls";
import type { CardEdit } from "./draft";
import { UNAVAILABLE_FOR_SCHEDULING } from "./status";
import type { ComputeActions } from "./use-compute";

type CardProps = { card: RegionCardState; limits: ComputeLimits; actions: ComputeActions };

export function RegionCard(props: CardProps) {
  return match(props.card.frame)
    .with({ type: "saved" }, (frame) => <SavedCard frame={frame} {...props} />)
    .with({ type: "new" }, ({ note }) => <NewCard note={note} {...props} />)
    .exhaustive();
}

type SavedFrame = Extract<RegionCardState["frame"], { type: "saved" }>;

function SavedCard({ frame, ...props }: CardProps & { frame: SavedFrame }) {
  const { actions } = props;
  const shown = regionInfo(frame.shown);
  return (
    <div
      className={cn(
        "overflow-hidden rounded-lg border transition-colors",
        frame.popIn && "animate-pop motion-reduce:animate-none",
        frame.highlighted ? "border-grayA-7" : "border-grayA-5",
      )}
    >
      <button
        type="button"
        aria-expanded={frame.open}
        onClick={() => actions.toggle(frame.name)}
        onMouseEnter={() => actions.hover(frame.name)}
        onMouseLeave={() => actions.hover(null)}
        className={cn(
          "group flex w-full min-w-0 items-center gap-3 py-3 pr-3 pl-4 transition-colors focus:outline-hidden focus-visible:bg-grayA-3",
          frame.highlighted && "bg-grayA-2",
        )}
      >
        <RectFlag region={shown.name} />
        <span className="flex min-w-0 flex-1 flex-col gap-0.5 text-left">
          <span className="flex items-center gap-2">
            <span className="text-sm font-medium text-gray-12">{shown.city}</span>
            <span className="font-mono text-xs text-gray-10">{shown.name}</span>
            {frame.unavailable ? <UnavailableChip /> : null}
          </span>
          <span className="truncate text-xs text-gray-11">{frame.summary}</span>
        </span>
        <span className="flex size-7 shrink-0 items-center justify-center rounded-md text-gray-10 transition-colors group-hover:bg-grayA-3 group-hover:text-gray-12">
          <IconChevronExpandYOutline12 className="size-3" />
        </span>
      </button>
      {frame.open ? <CardBody {...props} /> : null}
    </div>
  );
}

function scrollIntoViewOnMount(node: HTMLDivElement | null) {
  if (!node) {
    return;
  }
  const reduce = window.matchMedia("(prefers-reduced-motion: reduce)").matches;
  node.scrollIntoView({ behavior: reduce ? "auto" : "smooth", block: "start" });
}

function NewCard({ note, ...props }: CardProps & { note: string | null }) {
  return (
    <div
      ref={scrollIntoViewOnMount}
      className="animate-pop scroll-mt-20 overflow-hidden rounded-lg border border-dashed border-grayA-7 motion-reduce:animate-none"
    >
      <div className="flex flex-col gap-0.5 px-4 py-3">
        <span className="text-sm font-medium text-gray-12">New region</span>
        {note ? <span className="text-xs text-gray-11">{note}</span> : null}
      </div>
      <CardBody {...props} />
    </div>
  );
}

function CardBody({ card, limits, actions }: CardProps) {
  const controls = {
    view: card.view,
    limits,
    onEdit: (patch: CardEdit) => actions.edit(card.slot, patch),
  };
  return (
    <div className="divide-y divide-grayA-4 border-t border-grayA-4">
      <Row title="Region">
        <RegionSelect card={card} onPick={(region) => controls.onEdit({ region })} />
      </Row>
      <Row title="Size" description="CPU and memory per instance.">
        <SizeTrigger {...controls} />
      </Row>
      <Row
        title="Instances"
        description={`Scales between min and max based on CPU usage. Up to ${limits.replicas} per region.`}
      >
        <InstanceRange {...controls} />
      </Row>
      <Row title="Storage" description="Ephemeral disk per instance.">
        <StorageSelect {...controls} />
      </Row>
      <div className="flex items-center justify-between gap-3 bg-grayA-2 px-5 py-3">
        <Leading leading={card.footer.leading} actions={actions} />
        <SaveArea
          note={card.footer.note}
          save={card.footer.save}
          onSave={() => actions.save(card)}
        />
      </div>
    </div>
  );
}

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
    <SettingsRow className="gap-3 lg:flex-row lg:gap-6">
      <SettingsRowHeader className="lg:w-[34%]">
        <SettingsRowTitle>{title}</SettingsRowTitle>
        {description ? <SettingsRowDescription>{description}</SettingsRowDescription> : null}
      </SettingsRowHeader>
      <SettingsRowContent>{children}</SettingsRowContent>
    </SettingsRow>
  );
}

function Leading({ leading, actions }: { leading: FooterLeading; actions: ComputeActions }) {
  return match(leading)
    .with({ type: "remove" }, ({ name }) => (
      <RemoveRegionButton name={name} onRemove={() => actions.remove(name)} />
    ))
    .with({ type: "keep-one" }, () => (
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
    ))
    .with({ type: "cancel" }, () => (
      <Button variant="ghost" size="sm" onClick={actions.cancelAdd}>
        Cancel
      </Button>
    ))
    .exhaustive();
}

const NOTE_TONE = {
  error: "text-error-11",
  warning: "text-warning-11",
  muted: "text-gray-11",
} satisfies Record<FooterNote["tone"], string>;

function SaveArea({
  note,
  save,
  onSave,
}: {
  note: FooterNote | null;
  save: SaveButton;
  onSave: () => void;
}) {
  if (save.type === "none" && note === null) {
    return null;
  }
  return (
    <div className="flex items-center gap-3">
      {note ? <span className={cn("text-xs", NOTE_TONE[note.tone])}>{note.text}</span> : null}
      <SaveChangesButton save={save} onSave={onSave} />
    </div>
  );
}

const SAVE_BUTTON = {
  disabled: { disabled: true, loading: false },
  ready: { disabled: false, loading: false },
  saving: { disabled: true, loading: true },
} satisfies Record<Exclude<SaveButton["type"], "none">, { disabled: boolean; loading: boolean }>;

function SaveChangesButton({ save, onSave }: { save: SaveButton; onSave: () => void }) {
  if (save.type === "none") {
    return null;
  }
  return (
    <Button
      variant="primary"
      size="sm"
      className="px-3"
      {...SAVE_BUTTON[save.type]}
      onClick={onSave}
    >
      Save changes
    </Button>
  );
}

function RemoveRegionButton({ name, onRemove }: { name: string; onRemove: () => void }) {
  const [confirming, setConfirming] = useState(false);
  const region = regionInfo(name);
  return (
    <>
      <Button variant="outline" color="danger" size="sm" onClick={() => setConfirming(true)}>
        Remove region
      </Button>
      <AlertDialog open={confirming} onOpenChange={setConfirming}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Remove {region.city}?</AlertDialogTitle>
            <AlertDialogDescription>
              Instances in {region.name} stop on the next deployment. Traffic moves to your other
              regions.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction color="danger" onClick={onRemove}>
              Remove region
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  );
}

function UnavailableChip() {
  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger
          render={
            <span className="rounded-sm border border-warning-6 bg-warning-3 px-1 text-2xs leading-4 text-warning-11" />
          }
        >
          Unavailable
        </TooltipTrigger>
        <TooltipContent>{UNAVAILABLE_FOR_SCHEDULING}</TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
}

function RegionSelect({
  card,
  onPick,
}: {
  card: RegionCardState;
  onPick: (region: string) => void;
}) {
  const name = card.view.region;
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
        leftIcon={current ? <RectFlag region={current.name} size="sm" /> : undefined}
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
        {card.options.map((option) => {
          const region = regionInfo(option.name);
          return (
            <SelectItem key={option.name} value={option.name} disabled={option.disabled}>
              <span className="flex w-full items-center gap-2">
                <RectFlag region={region.name} size="sm" />
                {region.city}
                <span className="font-mono text-xs text-gray-10">{region.name}</span>
                {option.canSchedule ? null : (
                  <span className="ml-auto text-2xs text-gray-10">Unavailable</span>
                )}
              </span>
            </SelectItem>
          );
        })}
      </SelectContent>
    </Select>
  );
}
