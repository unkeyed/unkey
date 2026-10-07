"use client";

import { PlansScreen } from "@/app/(app)/[workspaceSlug]/settings/billing/components/plans-screen";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { planName } from "@/lib/billing/plan-card-state";
import {
  CUSTOM_SIZE_LABEL,
  type ComputeLimits,
  PRESETS,
  type Preset,
  STORAGE_OPTIONS,
  type SizeUnlock,
  type UnitField,
  formatUnit,
  sizeChoice,
  sizeOptions,
  unitFields,
  unitOptions,
} from "@/lib/compute/sizing";
import { formatCpu, formatMemory, formatStorage } from "@/lib/utils/deployment-formatters";
import {
  IconChevronExpandYOutline12,
  IconHardDriveOutline18,
  IconLockOutline12,
  IconMicrochipOutline18,
  IconMinusOutline12,
  IconPlusOutline12,
  IconRamOutline18,
} from "@unkey/icons";
import { match } from "@unkey/match";
import {
  NativeSelect,
  NativeSelectOption,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@unkey/ui";
import { useState } from "react";
import type { CardEdit, CardView } from "./draft";

export type ControlProps = {
  view: CardView;
  limits: ComputeLimits;
  onEdit: (patch: CardEdit) => void;
};

const CUSTOM_VALUE = "custom";

function pickSize(onEdit: ControlProps["onEdit"], value: string) {
  const preset = PRESETS.find((p) => p.id === value);
  onEdit(
    preset
      ? {
          sizeMode: "preset",
          size: { cpuMillicores: preset.cpuMillicores, memoryMib: preset.memoryMib },
        }
      : { sizeMode: "custom" },
  );
}

function Spec({ preset }: { preset: Pick<Preset, "cpuMillicores" | "memoryMib"> }) {
  return (
    <span className="flex items-center gap-2.5 text-xs text-gray-11">
      <span className="flex items-center gap-1">
        <IconMicrochipOutline18 className="size-3.5 text-gray-10" />
        {formatCpu(preset.cpuMillicores)}
      </span>
      <span className="flex items-center gap-1">
        <IconRamOutline18 className="size-3.5 text-gray-10" />
        {formatMemory(preset.memoryMib)}
      </span>
    </span>
  );
}

function unlockLabel(unlock: SizeUnlock): string {
  return match(unlock)
    .with({ type: "plan" }, ({ plan }) => planName(plan))
    .with({ type: "contact" }, () => "Contact us")
    .exhaustive();
}

export function SizeTrigger({ view, limits, onEdit }: ControlProps) {
  const choice = sizeChoice(view);
  const [plansOpen, setPlansOpen] = useState(false);
  return (
    <div className="flex flex-col gap-3">
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
            <span className="rounded border border-grayA-5 bg-grayA-3 px-1.5 py-0.5 font-mono text-xs font-medium text-gray-12">
              {choice.type === "preset" ? choice.preset.label : CUSTOM_SIZE_LABEL}
            </span>
            <Spec preset={view} />
          </span>
          <IconChevronExpandYOutline12 className="size-3 text-gray-10" />
        </DropdownMenuTrigger>
        <DropdownMenuContent align="start" className="w-(--anchor-width) min-w-80 p-1">
          <DropdownMenuRadioGroup
            value={choice.type === "preset" ? choice.preset.id : CUSTOM_VALUE}
            onValueChange={(next) => {
              if (typeof next === "string") {
                pickSize(onEdit, next);
              }
            }}
          >
            {sizeOptions(limits).map((option) =>
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
                  <Spec preset={option.preset} />
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
                  <Spec preset={option.preset} />
                  <span className="ml-auto flex items-center gap-1 text-2xs text-gray-11">
                    <IconLockOutline12 className="size-3" />
                    {unlockLabel(option.unlock)}
                  </span>
                </DropdownMenuItem>
              ),
            )}
            <DropdownMenuSeparator className="mx-0 my-1" />
            <DropdownMenuRadioItem
              value={CUSTOM_VALUE}
              closeOnClick
              className="gap-3 px-2 py-1.5 text-xs"
            >
              <span className="w-10 font-medium text-gray-12">{CUSTOM_SIZE_LABEL}</span>
              <span className="text-gray-11">Set CPU and memory yourself</span>
            </DropdownMenuRadioItem>
          </DropdownMenuRadioGroup>
        </DropdownMenuContent>
      </DropdownMenu>
      {choice.type === "custom" ? <CustomSize view={view} limits={limits} onEdit={onEdit} /> : null}
      <PlansScreen open={plansOpen} onOpenChange={setPlansOpen} reason="compute-size" />
    </div>
  );
}

function UnitSelect({
  label,
  field,
  value,
  onChange,
}: {
  label: string;
  field: UnitField;
  value: number | null;
  onChange: (value: number) => void;
}) {
  return (
    <NativeSelect
      aria-label={label}
      size="sm"
      wrapperClassName="w-full"
      value={value === null ? "" : String(value)}
      onChange={(e) => {
        const next = Number(e.target.value);
        if (Number.isFinite(next) && next > 0) {
          onChange(next);
        }
      }}
    >
      {value === null ? (
        <NativeSelectOption value="" disabled>
          Select
        </NativeSelectOption>
      ) : null}
      {unitOptions(field, value).map((option) => (
        <NativeSelectOption key={option} value={String(option)}>
          {formatUnit(option, field)}
        </NativeSelectOption>
      ))}
    </NativeSelect>
  );
}

function CustomSize({ view, limits, onEdit }: ControlProps) {
  const fields = unitFields(limits);
  const rows = [
    {
      field: fields.cpu,
      value: view.cpuMillicores,
      size: (cpuMillicores: number) => ({ cpuMillicores, memoryMib: view.memoryMib }),
    },
    {
      field: fields.memory,
      value: view.memoryMib,
      size: (memoryMib: number) => ({ cpuMillicores: view.cpuMillicores, memoryMib }),
    },
  ];
  return (
    <div className="flex flex-col gap-2">
      {rows.map(({ field, value, size }) => (
        <div key={field.label} className="flex items-center justify-between">
          <span className="text-xs text-gray-11">{field.label}</span>
          <div className="w-40">
            <UnitSelect
              label={field.label}
              field={field}
              value={value}
              onChange={(next) => onEdit({ size: size(next) })}
            />
          </div>
        </div>
      ))}
      <p className="text-xs text-gray-10">
        Max {formatUnit(fields.cpu.max, fields.cpu)} ·{" "}
        {formatUnit(fields.memory.max, fields.memory)} on your plan.
      </p>
    </div>
  );
}

function Stepper({
  label,
  value,
  current,
  min,
  max,
  onChange,
}: {
  label: string;
  value: number | null;
  current: number;
  min: number;
  max: number;
  onChange: (n: number) => void;
}) {
  return (
    <div className="flex items-center justify-between">
      <span className="text-xs text-gray-11">{label}</span>
      <div className="flex h-8 items-center rounded-md border border-grayA-5 bg-raised">
        <button
          type="button"
          aria-label={`Decrease ${label.toLowerCase()}`}
          disabled={value !== null && current <= min}
          onClick={() => onChange(Math.max(min, current - 1))}
          className="flex h-full w-8 items-center justify-center text-gray-11 transition-colors hover:text-gray-12 disabled:opacity-40"
        >
          <IconMinusOutline12 className="size-3" />
        </button>
        <span className="w-8 text-center font-mono text-sm tabular-nums text-gray-12">
          {value ?? "–"}
        </span>
        <button
          type="button"
          aria-label={`Increase ${label.toLowerCase()}`}
          disabled={value !== null && current >= max}
          onClick={() => onChange(Math.min(max, current + 1))}
          className="flex h-full w-8 items-center justify-center text-gray-11 transition-colors hover:text-gray-12 disabled:opacity-40"
        >
          <IconPlusOutline12 className="size-3" />
        </button>
      </div>
    </div>
  );
}

export function InstanceRange({ view, limits, onEdit }: ControlProps) {
  const { replicas } = view;
  const limit = limits.replicas;
  const setRange = (min: number, max: number) => onEdit({ replicas: { min, max } });
  const range = replicas.kind === "uniform" ? replicas : replicas.first;
  const shown = replicas.kind === "uniform";
  return (
    <div className="flex flex-col gap-2">
      <Stepper
        label="Minimum"
        value={shown ? range.min : null}
        current={range.min}
        min={1}
        max={limit}
        onChange={(n) => setRange(n, Math.max(n, range.max))}
      />
      <Stepper
        label="Maximum"
        value={shown ? range.max : null}
        current={range.max}
        min={1}
        max={limit}
        onChange={(n) => setRange(Math.min(n, range.min), n)}
      />
      {replicas.kind === "mixed" ? (
        <p className="text-xs text-warning-11">
          Regions use different ranges. Setting a range here applies it to every region.
        </p>
      ) : null}
    </div>
  );
}

export function StorageSelect({ view, limits, onEdit }: ControlProps) {
  const options = STORAGE_OPTIONS.filter((mib) => mib <= limits.storageMib);
  const custom = view.storageMode === "custom";
  const field = unitFields(limits).storage;
  return (
    <div className="flex flex-col gap-2">
      <Select
        value={custom ? CUSTOM_VALUE : String(view.storageMib)}
        onValueChange={(next) => {
          if (next === CUSTOM_VALUE) {
            onEdit({ storageMode: "custom" });
            return;
          }
          const storageMib = Number(next);
          if (Number.isFinite(storageMib)) {
            onEdit({ storageMode: "preset", storageMib });
          }
        }}
      >
        <SelectTrigger
          wrapperClassName="w-full"
          leftIcon={<IconHardDriveOutline18 className="size-3.5 text-gray-11" />}
          aria-label="Ephemeral storage"
        >
          <SelectValue>{custom ? CUSTOM_SIZE_LABEL : formatStorage(view.storageMib)}</SelectValue>
        </SelectTrigger>
        <SelectContent>
          {options.map((mib) => (
            <SelectItem key={mib} value={String(mib)}>
              {formatStorage(mib)}
            </SelectItem>
          ))}
          <SelectItem value={CUSTOM_VALUE}>Custom…</SelectItem>
        </SelectContent>
      </Select>
      {custom ? (
        <div className="flex flex-col gap-1">
          <UnitSelect
            label="Custom storage"
            field={field}
            value={view.storageMib > 0 ? view.storageMib : null}
            onChange={(storageMib) => onEdit({ storageMib })}
          />
          <p className="text-xs text-gray-10">Up to {formatUnit(field.max, field)} per instance.</p>
        </div>
      ) : null}
    </div>
  );
}
