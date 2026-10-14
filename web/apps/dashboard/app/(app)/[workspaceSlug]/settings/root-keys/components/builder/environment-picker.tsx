"use client";

import {
  AppEnvironmentList,
  type AppEnvironmentListProps,
} from "@/components/deploy/app-environment-list";
import { getSelectedEnvironmentIds } from "@/components/deploy/app-environment-selection";
import { IconChevronExpandYOutline18, IconMagnifierOutline12 } from "@unkey/icons";
import { Checkbox, Popover, PopoverContent, PopoverTrigger } from "@unkey/ui";
import { useId, useState } from "react";
import { ALL_INSTANCES } from "./lib/catalogue.types";

type EnvironmentPickerProps = Pick<
  AppEnvironmentListProps,
  "apps" | "environments" | "isLoading"
> & {
  value: string[];
  onChange: (value: string[]) => void;
};

export function EnvironmentPicker({
  apps,
  environments,
  isLoading,
  value,
  onChange,
}: EnvironmentPickerProps) {
  const id = useId();
  const [search, setSearch] = useState("");
  const allSelected = value.includes(ALL_INSTANCES);

  return (
    <Popover onOpenChange={() => setSearch("")}>
      <PopoverTrigger
        aria-label="Select environments"
        className="flex h-9 w-full min-w-0 items-center gap-2 rounded-lg border bg-raised px-3 text-sm leading-5 text-grayA-12 transition-colors duration-300 hover:border-strong"
      >
        <span className="truncate">
          {allSelected ? (
            "All environments"
          ) : value.length === 0 ? (
            <span className="text-grayA-8">Select environments…</span>
          ) : (
            `${value.length} ${value.length === 1 ? "environment" : "environments"}`
          )}
        </span>
        <IconChevronExpandYOutline18 className="ml-auto shrink-0 text-grayA-9" />
      </PopoverTrigger>
      <PopoverContent
        align="start"
        className="w-80 max-w-(--available-width) overflow-hidden p-0"
        aria-label="Select environments"
      >
        <div className="flex items-center gap-2 border-b px-3 py-2">
          <IconMagnifierOutline12 className="shrink-0 text-gray-9" />
          <input
            value={search}
            onChange={(event) => setSearch(event.target.value)}
            aria-label="Search environments"
            placeholder="Search environments…"
            className="h-6 w-full bg-transparent text-sm text-gray-12 outline-hidden placeholder:text-gray-9"
          />
        </div>
        <div className="border-b p-2">
          <label
            htmlFor={id}
            className="flex h-9 cursor-pointer items-center gap-2 rounded-md px-2 hover:bg-grayA-3"
          >
            <Checkbox
              id={id}
              checked={allSelected}
              onCheckedChange={(checked) => onChange(checked ? [ALL_INSTANCES] : [])}
              className="size-4 shrink-0 rounded-sm [&_svg]:size-3"
            />
            <span className="text-xs font-medium text-gray-12">All environments</span>
          </label>
        </div>
        <AppEnvironmentList
          apps={apps}
          environments={environments}
          isLoading={isLoading}
          search={search}
          value={{ appIds: new Set(), environmentIds: new Set(allSelected ? [] : value) }}
          onChange={(selection) => onChange(getSelectedEnvironmentIds(selection, environments))}
        />
      </PopoverContent>
    </Popover>
  );
}
