"use client";

import { ResourceSearchInput } from "@/components/resource-search-input";
import { ToolbarSelect } from "@/components/toolbar-select";
import { IconBarsFilterOutline18, IconLayers3Outline18 } from "@unkey/icons";
import { ResourceListHeader } from "@unkey/ui";
import { EnvironmentSelect } from "../../environment-select";

export const ENV_VARS_SEARCH_KEY = "search";

const SORT_OPTIONS = [
  { value: "last-updated", label: "Last updated" },
  { value: "name-asc", label: "Name" },
] as const;
export type SortOption = (typeof SORT_OPTIONS)[number]["value"];

type EnvVarsToolbarProps = {
  environmentFilter: string;
  onEnvironmentFilterChange: (value: string) => void;
  sortBy: SortOption;
  onSortChange: (value: SortOption) => void;
};

export function EnvVarsToolbar({
  environmentFilter,
  onEnvironmentFilterChange,
  sortBy,
  onSortChange,
}: EnvVarsToolbarProps) {
  return (
    <ResourceListHeader>
      <ResourceSearchInput
        queryKey={ENV_VARS_SEARCH_KEY}
        label="Search environment variables"
        placeholder="Search by key"
      />
      <EnvironmentSelect
        value={environmentFilter}
        onValueChange={onEnvironmentFilterChange}
        wrapperClassName="md:w-46"
        className="h-8 w-full bg-gray-1"
        leftIcon={<IconLayers3Outline18 className="size-3.5 text-gray-9" />}
      />
      <ToolbarSelect
        label="Sort"
        icon={<IconBarsFilterOutline18 className="size-3.5 text-gray-9" />}
        value={sortBy}
        options={SORT_OPTIONS}
        onChange={onSortChange}
      />
    </ResourceListHeader>
  );
}
