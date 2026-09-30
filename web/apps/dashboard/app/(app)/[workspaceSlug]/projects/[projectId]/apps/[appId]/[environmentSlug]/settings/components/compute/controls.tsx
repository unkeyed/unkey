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
  IconChevronExpandYOutline12,
  IconHardDriveOutline18,
  IconLockOutline12,
  IconMicrochipOutline18,
  IconMinusOutline12,
  IconPlusOutline12,
  IconRamOutline18,
} from "@unkey/icons";
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
import {
  PRESETS,
  type Preset,
  STORAGE_OPTIONS,
  type UnitField,
  activePreset,
  formatCpu,
  formatMemory,
  formatStorage,
  formatUnit,
  planForPreset,
  presetFits,
  unitFields,
  unitOptions,
} from "./model";
import type { ComputeCard } from "./use-compute";

const CUSTOM = "custom";

function pickSize(c: ComputeCard, id: string) {
  const preset = PRESETS.find((p) => p.id === id);
  c.edit(
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

export function SizeTrigger({ c }: { c: ComputeCard }) {
  const value = activePreset(c.view)?.id ?? CUSTOM;
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
              {activePreset(c.view)?.label ?? "Custom"}
            </span>
            <Spec preset={c.view} />
          </span>
          <IconChevronExpandYOutline12 className="size-3 text-gray-10" />
        </DropdownMenuTrigger>
        <DropdownMenuContent align="start" className="w-(--anchor-width) min-w-80 p-1">
          <DropdownMenuRadioGroup
            value={value}
            onValueChange={(next) => {
              if (typeof next === "string") {
                pickSize(c, next);
              }
            }}
          >
            {PRESETS.map((p) => {
              if (presetFits(p, c.limits)) {
                return (
                  <DropdownMenuRadioItem
                    key={p.id}
                    value={p.id}
                    closeOnClick
                    className="gap-3 px-2 py-1.5"
                  >
                    <span className="w-10 font-mono text-xs font-medium text-gray-12">
                      {p.label}
                    </span>
                    <Spec preset={p} />
                  </DropdownMenuRadioItem>
                );
              }
              const plan = planForPreset(p);
              return (
                <DropdownMenuItem
                  key={p.id}
                  onClick={() => setPlansOpen(true)}
                  className="gap-3 px-2 py-1.5 text-gray-11"
                >
                  <span className="w-10 font-mono text-xs font-medium text-gray-11">{p.label}</span>
                  <Spec preset={p} />
                  <span className="ml-auto flex items-center gap-1 text-[11px] text-gray-11">
                    <IconLockOutline12 className="size-3" />
                    {plan ? planName(plan, undefined) : "Contact us"}
                  </span>
                </DropdownMenuItem>
              );
            })}
            <DropdownMenuSeparator className="mx-0 my-1" />
            <DropdownMenuRadioItem
              value={CUSTOM}
              closeOnClick
              className="gap-3 px-2 py-1.5 text-xs"
            >
              <span className="w-10 font-medium text-gray-12">Custom</span>
              <span className="text-gray-11">Set CPU and memory yourself</span>
            </DropdownMenuRadioItem>
          </DropdownMenuRadioGroup>
        </DropdownMenuContent>
      </DropdownMenu>
      {value === CUSTOM ? <CustomSize c={c} /> : null}
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
          {formatUnit(option, field)} {field.unit}
        </NativeSelectOption>
      ))}
    </NativeSelect>
  );
}

function CustomSize({ c }: { c: ComputeCard }) {
  const fields = unitFields(c.limits);
  return (
    <div className="flex flex-col gap-2">
      <div className="flex items-center justify-between">
        <span className="text-xs text-gray-11">CPU</span>
        <div className="w-40">
          <UnitSelect
            label="CPU"
            field={fields.cpu}
            value={c.view.cpuMillicores}
            onChange={(cpuMillicores) =>
              c.edit({ size: { cpuMillicores, memoryMib: c.view.memoryMib } })
            }
          />
        </div>
      </div>
      <div className="flex items-center justify-between">
        <span className="text-xs text-gray-11">Memory</span>
        <div className="w-40">
          <UnitSelect
            label="Memory"
            field={fields.memory}
            value={c.view.memoryMib}
            onChange={(memoryMib) =>
              c.edit({ size: { cpuMillicores: c.view.cpuMillicores, memoryMib } })
            }
          />
        </div>
      </div>
      <p className="text-xs text-gray-10">
        Max {formatUnit(fields.cpu.max, fields.cpu)} vCPU ·{" "}
        {formatUnit(fields.memory.max, fields.memory)} GiB on your plan.
      </p>
    </div>
  );
}

function Stepper({
  label,
  value,
  min,
  max,
  onChange,
}: {
  label: string;
  value: number | null;
  min: number;
  max: number;
  onChange: (n: number) => void;
}) {
  const current = value ?? min;
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

export function InstanceRange({ c }: { c: ComputeCard }) {
  const { replicas } = c.view;
  const limit = c.limits.replicas;
  const setRange = (min: number, max: number) => c.edit({ replicas: { min, max } });
  const min = replicas.kind === "uniform" ? replicas.min : null;
  const max = replicas.kind === "uniform" ? replicas.max : null;
  return (
    <div className="flex flex-col gap-2">
      <Stepper
        label="Minimum"
        value={min}
        min={1}
        max={limit}
        onChange={(n) => setRange(n, Math.max(n, max ?? n))}
      />
      <Stepper
        label="Maximum"
        value={max}
        min={1}
        max={limit}
        onChange={(n) => setRange(Math.min(n, min ?? n), n)}
      />
      {replicas.kind === "mixed" ? (
        <p className="text-xs text-warning-11">
          Regions use different ranges. Setting a range here applies it to every region.
        </p>
      ) : null}
    </div>
  );
}

const CUSTOM_STORAGE = "custom";

export function StorageSelect({ c }: { c: ComputeCard }) {
  const options = STORAGE_OPTIONS.filter((mib) => mib <= c.limits.storageMib);
  const custom = c.view.storageMode === "custom";
  const field = unitFields(c.limits).storage;
  return (
    <div className="flex flex-col gap-2">
      <Select
        value={custom ? CUSTOM_STORAGE : String(c.view.storageMib)}
        onValueChange={(next) => {
          if (next === CUSTOM_STORAGE) {
            c.edit({ storageMode: "custom" });
            return;
          }
          const storageMib = Number(next);
          if (Number.isFinite(storageMib)) {
            c.edit({ storageMode: "preset", storageMib });
          }
        }}
      >
        <SelectTrigger
          wrapperClassName="w-full"
          leftIcon={<IconHardDriveOutline18 className="size-3.5 text-gray-11" />}
          aria-label="Ephemeral storage"
        >
          <SelectValue>{custom ? "Custom" : formatStorage(c.view.storageMib)}</SelectValue>
        </SelectTrigger>
        <SelectContent>
          {options.map((mib) => (
            <SelectItem key={mib} value={String(mib)}>
              {formatStorage(mib)}
            </SelectItem>
          ))}
          <SelectItem value={CUSTOM_STORAGE}>Custom…</SelectItem>
        </SelectContent>
      </Select>
      {custom ? (
        <div className="flex flex-col gap-1">
          <UnitSelect
            label="Custom storage"
            field={field}
            value={c.view.storageMib > 0 ? c.view.storageMib : null}
            onChange={(storageMib) => c.edit({ storageMib })}
          />
          <p className="text-xs text-gray-10">
            Up to {formatUnit(field.max, field)} GiB per instance.
          </p>
        </div>
      ) : null}
    </div>
  );
}
