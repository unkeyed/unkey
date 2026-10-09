"use client";

import { PlansScreen } from "@/app/(app)/[workspaceSlug]/settings/billing/components/plans-screen";
import { RegionFlag } from "@/components/region-flag";
import { planName } from "@/lib/billing/plan-card-state";
import {
  CUSTOM_SIZE_LABEL,
  type Preset,
  type SizeUnlock,
  type UnitField,
  sizeOptions,
  unitFields,
  unitOptions,
} from "@/lib/compute/sizing";
import { regionInfo } from "@/lib/regions";
import { formatCpu, formatMemory, formatStorage } from "@/lib/utils/deployment-formatters";
import {
  IconHardDriveOutline18,
  IconLockOutline12,
  IconMicrochipOutline18,
  IconRamOutline18,
} from "@unkey/icons";
import { match } from "@unkey/match";
import {
  Badge,
  FormField,
  Select,
  SelectContent,
  SelectItem,
  SelectSeparator,
  SelectTrigger,
  SelectValue,
} from "@unkey/ui";
import { cn } from "cn";
import { useState } from "react";
import type { Replicas } from "./draft";
import { REGION_STATES, regionChoices, regionsNote } from "./status";
import type { ComputeForm } from "./use-compute";
import { CUSTOM_VALUE, useSizeChoice } from "./use-size-choice";
import { useStorageChoice } from "./use-storage-choice";

type FieldProps = { compute: ComputeForm };

export function ComputeFields({ compute, className }: FieldProps & { className?: string }) {
  return (
    <div className={cn("grid gap-6", className)}>
      <RegionsField compute={compute} />
      <SizeField compute={compute} />
      <InstancesField compute={compute} />
      <StorageField compute={compute} />
    </div>
  );
}

function RegionsField({ compute }: FieldProps) {
  const { values, available, busy } = compute;
  const choices = regionChoices(available, values.regions);
  const states = new Map(choices.map((choice) => [choice.name, REGION_STATES[choice.state]]));
  return (
    <FormField label="Regions" description={regionsNote(available)} className="col-span-full">
      {({ id, describedBy }) => (
        <Select
          multiple
          value={values.regions}
          disabled={busy}
          onValueChange={(next) => compute.change("regions", next)}
        >
          <SelectTrigger id={id} aria-describedby={describedBy} className="overflow-hidden">
            <SelectValue className="gap-1.5">
              {values.regions.map((name) => (
                <span
                  key={name}
                  data-warning={states.get(name)?.warning}
                  className="flex shrink-0 items-center gap-1.5 rounded border border-grayA-5 bg-grayA-3 py-0.5 pr-1.5 pl-1 text-xs font-medium text-gray-12 data-[warning=true]:border-warning-7 data-[warning=true]:text-warning-11"
                >
                  <RegionFlag region={name} shape="rect" size="sm" />
                  {regionInfo(name).city}
                </span>
              ))}
            </SelectValue>
          </SelectTrigger>
          <SelectContent>
            {choices.map((choice) => {
              const state = REGION_STATES[choice.state];
              return (
                <SelectItem key={choice.name} value={choice.name} disabled={state.locked}>
                  <span className="flex w-full items-center gap-2">
                    <RegionFlag region={choice.name} shape="rect" size="sm" />
                    <span className="font-medium">{regionInfo(choice.name).city}</span>
                    <span className="font-mono text-xs text-gray-10">{choice.name}</span>
                    {state.hint ? (
                      <span className="ml-auto pl-3 text-2xs text-gray-10">{state.hint}</span>
                    ) : null}
                  </span>
                </SelectItem>
              );
            })}
          </SelectContent>
        </Select>
      )}
    </FormField>
  );
}

function PresetSpec({ preset }: { preset: Pick<Preset, "cpuMillicores" | "memoryMib"> }) {
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

function SizeField({ compute }: FieldProps) {
  const { values, limits, busy } = compute;
  const size = useSizeChoice({ values, onChange: compute.change });
  const [plansOpen, setPlansOpen] = useState(false);
  const fields = unitFields(limits);
  const options = sizeOptions(limits);
  const customRows = [
    { key: "cpuMillicores", field: fields.cpu },
    { key: "memoryMib", field: fields.memory },
  ] as const;
  const pick = (next: string) => {
    if (options.some((option) => option.type === "locked" && option.preset.id === next)) {
      setPlansOpen(true);
      return;
    }
    size.pick(next);
  };
  return (
    <FormField
      label="Size"
      description={`CPU and memory per instance. Up to ${fields.cpu.format(fields.cpu.max)} · ${fields.memory.format(fields.memory.max)} on your plan.`}
    >
      {({ id, describedBy }) => (
        <div className="flex flex-col gap-3">
          <Select
            value={size.selected}
            disabled={busy}
            onValueChange={(next) => {
              if (typeof next === "string") {
                pick(next);
              }
            }}
          >
            <SelectTrigger id={id} aria-describedby={describedBy}>
              <SelectValue className="gap-3">
                <Badge variant="code">{size.label}</Badge>
                <PresetSpec preset={values} />
              </SelectValue>
            </SelectTrigger>
            <SelectContent>
              {options.map((option) => (
                <SelectItem key={option.preset.id} value={option.preset.id}>
                  <span className="flex w-full items-center gap-3">
                    <span className="flex w-12">
                      <Badge variant="code">{option.preset.label}</Badge>
                    </span>
                    <PresetSpec preset={option.preset} />
                    {option.type === "locked" ? (
                      <span className="ml-auto flex items-center gap-1 pl-3 text-2xs text-gray-11">
                        <IconLockOutline12 className="size-3" />
                        {unlockLabel(option.unlock)}
                      </span>
                    ) : null}
                  </span>
                </SelectItem>
              ))}
              <SelectSeparator />
              <SelectItem value={CUSTOM_VALUE}>
                <span className="flex w-full items-center gap-3 text-xs">
                  <span className="flex w-12">
                    <Badge variant="code">{CUSTOM_SIZE_LABEL}</Badge>
                  </span>
                  <span className="text-gray-11">Set CPU and memory yourself</span>
                </span>
              </SelectItem>
            </SelectContent>
          </Select>
          {size.choice.type === "custom" ? (
            <div className="grid grid-cols-2 gap-3">
              {customRows.map(({ key, field }) => (
                <div key={key} className="flex flex-col gap-1">
                  <span className="text-xs font-medium text-gray-11">{field.label}</span>
                  <UnitSelect
                    field={field}
                    disabled={busy}
                    value={values[key]}
                    onChange={(next) => compute.change(key, next)}
                  />
                </div>
              ))}
            </div>
          ) : null}
          <PlansScreen open={plansOpen} onOpenChange={setPlansOpen} reason="compute-size" />
        </div>
      )}
    </FormField>
  );
}

function UnitSelect({
  field,
  value,
  disabled,
  onChange,
}: {
  field: UnitField;
  value: number | null;
  disabled?: boolean;
  onChange: (value: number) => void;
}) {
  return (
    <Select
      value={value === null ? null : String(value)}
      disabled={disabled}
      onValueChange={(next) => {
        const parsed = Number(next);
        if (Number.isFinite(parsed) && parsed > 0) {
          onChange(parsed);
        }
      }}
    >
      <SelectTrigger wrapperClassName="w-full" aria-label={field.label}>
        <SelectValue>
          {value === null ? <span className="text-gray-9">Select</span> : field.format(value)}
        </SelectValue>
      </SelectTrigger>
      <SelectContent>
        {unitOptions(field, value).map((option) => (
          <SelectItem key={option} value={String(option)}>
            {field.format(option)}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}

function InstancesField({ compute }: FieldProps) {
  const { values, limits, busy } = compute;
  const { min, max } = values.replicas;
  const counts = Array.from({ length: limits.replicas }, (_, i) => i + 1);
  const bounds: { key: keyof Replicas; label: string; next: (n: number) => Replicas }[] = [
    { key: "min", label: "Min", next: (n) => ({ min: n, max: Math.max(n, max) }) },
    { key: "max", label: "Max", next: (n) => ({ min: Math.min(n, min), max: n }) },
  ];
  return (
    <FormField
      label="Instances"
      description={`Scales between min and max on CPU usage. Up to ${limits.replicas} per region.`}
    >
      {({ describedBy }) => (
        <div className="grid grid-cols-2 gap-3">
          {bounds.map((bound) => (
            <Select
              key={bound.key}
              value={String(values.replicas[bound.key])}
              disabled={busy}
              onValueChange={(next) => {
                const n = Number(next);
                if (Number.isInteger(n) && n > 0) {
                  compute.change("replicas", bound.next(n));
                }
              }}
            >
              <SelectTrigger aria-label={`${bound.label} instances`} aria-describedby={describedBy}>
                <SelectValue>
                  <span className="mr-2 text-gray-10">{bound.label}</span>
                  <span className="font-mono tabular-nums">{values.replicas[bound.key]}</span>
                </SelectValue>
              </SelectTrigger>
              <SelectContent>
                {counts.map((n) => (
                  <SelectItem key={n} value={String(n)}>
                    {n}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          ))}
        </div>
      )}
    </FormField>
  );
}

function StorageField({ compute }: FieldProps) {
  const { values, limits, busy } = compute;
  const storage = useStorageChoice({ values, limits, onChange: compute.change });
  return (
    <FormField label="Storage" description="Ephemeral disk per instance.">
      {({ id, describedBy }) => (
        <div className="flex flex-col gap-3">
          <Select
            value={storage.selected}
            disabled={busy}
            onValueChange={(next) => {
              if (typeof next === "string") {
                storage.pick(next);
              }
            }}
          >
            <SelectTrigger
              id={id}
              aria-describedby={describedBy}
              leftIcon={<IconHardDriveOutline18 className="size-3.5 text-gray-11" />}
            >
              <SelectValue>{formatStorage(values.storageMib)}</SelectValue>
            </SelectTrigger>
            <SelectContent>
              {storage.options.map((mib) => (
                <SelectItem key={mib} value={String(mib)}>
                  {formatStorage(mib)}
                </SelectItem>
              ))}
              <SelectSeparator />
              <SelectItem value={CUSTOM_VALUE}>Custom…</SelectItem>
            </SelectContent>
          </Select>
          {storage.custom ? (
            <UnitSelect
              field={storage.field}
              disabled={busy}
              value={storage.customValue}
              onChange={(storageMib) => compute.change("storageMib", storageMib)}
            />
          ) : null}
        </div>
      )}
    </FormField>
  );
}
