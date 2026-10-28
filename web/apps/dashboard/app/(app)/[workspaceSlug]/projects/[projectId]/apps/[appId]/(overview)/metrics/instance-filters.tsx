"use client";

import { RegionFlag } from "@/components/region-flag";
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuSub,
  DropdownMenuSubContent,
  DropdownMenuSubTrigger,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { regionInfo } from "@/lib/regions";
import {
  IconBarsFilterOutline18,
  IconChartActivityOutline18,
  IconEarthOutline18,
  IconServerOutline18,
} from "@unkey/icons";
import { Button, ControlPill } from "@unkey/ui";
import type { ReactNode } from "react";
import {
  type FilterChoice,
  type FilterKey,
  type InstanceFilters,
  statusLabel,
} from "./instances-model";

type Field = {
  key: FilterKey;
  label: string;
  icon: ReactNode;
  choice: FilterChoice;
  name: (option: string) => string;
  option: (option: string) => ReactNode;
};

type OnChange = (key: FilterKey, values: string[]) => void;

function FieldOptions({ field, onChange }: { field: Field; onChange: OnChange }) {
  const { options, selected } = field.choice;
  return options.map((option) => (
    <DropdownMenuCheckboxItem
      key={option}
      checked={selected.includes(option)}
      onCheckedChange={(checked) =>
        onChange(
          field.key,
          options.filter((candidate) =>
            candidate === option ? checked : selected.includes(candidate),
          ),
        )
      }
    >
      {field.option(option)}
    </DropdownMenuCheckboxItem>
  ));
}

export function InstanceFilterBar({
  filters,
  onChange,
}: {
  filters: InstanceFilters;
  onChange: OnChange;
}) {
  const fields: Field[] = [
    {
      key: "regions",
      label: "Region",
      icon: <IconEarthOutline18 className="size-3.5 text-gray-9" />,
      choice: filters.regions,
      name: (region) => regionInfo(region).city,
      option: (region) => (
        <span className="flex items-center gap-2">
          <RegionFlag region={region} shape="rect" size="xs" />
          {regionInfo(region).city}
          <span className="font-mono text-xs opacity-60">{region}</span>
        </span>
      ),
    },
    {
      key: "statuses",
      label: "Status",
      icon: <IconChartActivityOutline18 className="size-3.5 text-gray-9" />,
      choice: filters.statuses,
      name: statusLabel,
      option: statusLabel,
    },
    {
      key: "instances",
      label: "Instance",
      icon: <IconServerOutline18 className="size-3.5 text-gray-9" />,
      choice: filters.instances,
      name: (key) => key,
      option: (key) => (
        <span className="flex items-center gap-2">
          <IconServerOutline18 className="size-3.5 text-gray-9" />
          <span className="font-mono text-xs">{key}</span>
        </span>
      ),
    },
  ];
  return (
    <div className="flex min-h-11 flex-wrap items-center gap-2 border-t border-grayA-4 px-3 py-1.5 text-xs">
      <DropdownMenu>
        <DropdownMenuTrigger
          render={<Button variant="outline" size="sm" className="h-7 gap-1.5 px-2" />}
        >
          <IconBarsFilterOutline18 className="size-3.5" />
          Add filter
        </DropdownMenuTrigger>
        <DropdownMenuContent align="start" className="w-44">
          {fields.map((field) => (
            <DropdownMenuSub key={field.key}>
              <DropdownMenuSubTrigger>
                {field.icon}
                {field.label}
                {field.choice.selected.length > 0 ? (
                  <span className="ml-auto text-xs tabular-nums text-gray-10">
                    {field.choice.selected.length}
                  </span>
                ) : null}
              </DropdownMenuSubTrigger>
              <DropdownMenuSubContent className="max-h-80 overflow-y-auto">
                <FieldOptions field={field} onChange={onChange} />
              </DropdownMenuSubContent>
            </DropdownMenuSub>
          ))}
        </DropdownMenuContent>
      </DropdownMenu>
      {fields
        .flatMap((field) => field.choice.selected.map((value) => ({ field, value })))
        .map(({ field, value }, index) => (
          <ControlPill
            key={`${field.key}:${value}`}
            variant="outline"
            index={index}
            filter={{ id: value, field: field.label, operator: "is", value }}
            formatFieldName={(name) => name}
            formatValue={(option) => field.name(String(option))}
            onRemove={() =>
              onChange(
                field.key,
                field.choice.selected.filter((v) => v !== value),
              )
            }
          />
        ))}
    </div>
  );
}
