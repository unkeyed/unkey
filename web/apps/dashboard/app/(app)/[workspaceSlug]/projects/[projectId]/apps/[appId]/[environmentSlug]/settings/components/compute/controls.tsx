"use client";

import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  IconChevronDownOutline12,
  IconChevronExpandYOutline12,
  IconHardDriveOutline18,
  IconLockOutline12,
  IconMicrochipOutline18,
  IconMinusOutline12,
  IconPlusOutline12,
  IconRamOutline18,
} from "@unkey/icons";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@unkey/ui";
import { useState } from "react";
import {
  PRESETS,
  type Preset,
  STORAGE_OPTIONS,
  activePreset,
  formatCpu,
  formatMemory,
  formatStorage,
  presetFits,
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
              const locked = !presetFits(p, c.limits);
              return (
                <DropdownMenuRadioItem
                  key={p.id}
                  value={p.id}
                  disabled={locked}
                  closeOnClick
                  className="gap-3 px-2 py-1.5"
                >
                  <span className="w-10 font-mono text-xs font-medium text-gray-12">{p.label}</span>
                  <Spec preset={p} />
                  {locked ? (
                    <span className="ml-auto flex items-center gap-1 text-[11px] text-gray-10">
                      <IconLockOutline12 className="size-3" />
                      Upgrade
                    </span>
                  ) : null}
                </DropdownMenuRadioItem>
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
    </div>
  );
}

const MIB_PER_GIB = 1024;

function trimNumber(n: number): string {
  return String(Math.round(n * 100) / 100);
}

function UnitInput({
  label,
  unit,
  value,
  step,
  min,
  max,
  onChange,
}: {
  label: string;
  unit: string;
  value: number | null;
  step: number;
  min: number;
  max: number;
  onChange: (n: number) => void;
}) {
  const [text, setText] = useState(value === null ? "" : trimNumber(value));
  const commit = (next: string) => {
    setText(next);
    const n = Number(next);
    if (next !== "" && Number.isFinite(n) && n >= min && n <= max) {
      onChange(n);
    }
  };
  const nudge = (direction: 1 | -1) => {
    const base = Number(text);
    const start = Number.isFinite(base) && text !== "" ? base : min;
    commit(trimNumber(Math.min(max, Math.max(min, start + direction * step))));
  };
  return (
    <div className="flex h-8 w-full items-stretch overflow-hidden rounded-md border border-grayA-5 bg-raised focus-within:border-grayA-8">
      <input
        type="text"
        inputMode="decimal"
        aria-label={label}
        value={text}
        onChange={(e) => commit(e.target.value)}
        onKeyDown={(e) => {
          if (e.key === "ArrowUp" || e.key === "ArrowDown") {
            e.preventDefault();
            nudge(e.key === "ArrowUp" ? 1 : -1);
          }
        }}
        className="min-w-0 flex-1 bg-transparent px-3 font-mono text-sm text-gray-12 outline-hidden placeholder:text-gray-9"
      />
      <div className="flex flex-col justify-center pr-1.5">
        <button
          type="button"
          tabIndex={-1}
          aria-label={`Increase ${label}`}
          onClick={() => nudge(1)}
          className="flex h-3.5 w-4 items-center justify-center text-gray-10 hover:text-gray-12"
        >
          <IconChevronDownOutline12 className="size-2.5 rotate-180" />
        </button>
        <button
          type="button"
          tabIndex={-1}
          aria-label={`Decrease ${label}`}
          onClick={() => nudge(-1)}
          className="flex h-3.5 w-4 items-center justify-center text-gray-10 hover:text-gray-12"
        >
          <IconChevronDownOutline12 className="size-2.5" />
        </button>
      </div>
      <span className="flex w-12 items-center justify-center border-l border-grayA-5 bg-grayA-2 text-xs text-gray-11">
        {unit}
      </span>
    </div>
  );
}

function CustomSize({ c }: { c: ComputeCard }) {
  const maxCpu = c.limits.cpuMillicores / 1000;
  const maxMemory = c.limits.memoryMib / MIB_PER_GIB;
  return (
    <div className="flex flex-col gap-2">
      <div className="flex items-center justify-between">
        <span className="text-xs text-gray-11">CPU</span>
        <div className="w-40">
          <UnitInput
            label="CPU"
            unit="vCPU"
            value={c.view.cpuMillicores / 1000}
            step={0.25}
            min={0.25}
            max={maxCpu}
            onChange={(v) =>
              c.edit({
                size: { cpuMillicores: Math.round(v * 1000), memoryMib: c.view.memoryMib },
              })
            }
          />
        </div>
      </div>
      <div className="flex items-center justify-between">
        <span className="text-xs text-gray-11">Memory</span>
        <div className="w-40">
          <UnitInput
            label="Memory"
            unit="GiB"
            value={c.view.memoryMib / MIB_PER_GIB}
            step={0.25}
            min={0.25}
            max={maxMemory}
            onChange={(v) =>
              c.edit({
                size: {
                  cpuMillicores: c.view.cpuMillicores,
                  memoryMib: Math.round(v * MIB_PER_GIB),
                },
              })
            }
          />
        </div>
      </div>
      <p className="text-xs text-gray-10">
        Max {trimNumber(maxCpu)} vCPU · {trimNumber(maxMemory)} GiB on your plan.
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
  const limitGib = c.limits.storageMib / MIB_PER_GIB;
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
          <UnitInput
            label="Custom storage"
            unit="GiB"
            value={c.view.storageMib > 0 ? c.view.storageMib / MIB_PER_GIB : null}
            step={0.5}
            min={0.5}
            max={limitGib}
            onChange={(v) => c.edit({ storageMib: Math.round(v * MIB_PER_GIB) })}
          />
          <p className="text-xs text-gray-10">Up to {limitGib} GiB per instance.</p>
        </div>
      ) : null}
    </div>
  );
}
