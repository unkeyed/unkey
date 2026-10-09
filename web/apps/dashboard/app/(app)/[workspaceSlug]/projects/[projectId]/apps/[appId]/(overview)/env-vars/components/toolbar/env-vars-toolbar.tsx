"use client";

import { ResourceSearchInput } from "@/components/resource-search-input";
import {
  IconBarsFilterOutline18,
  IconChevronDownOutline18,
  IconLayers3Outline18,
} from "@unkey/icons";
import {
  ResourceListHeader,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@unkey/ui";
import { EnvironmentSelect } from "../../../../../_components/env-vars/shared/environment-select";

export const ENV_VARS_SEARCH_KEY = "search";

const SORT_LABELS = {
  "last-updated": "Newest",
  "name-asc": "Name A-Z",
} as const;
export type SortOption = keyof typeof SORT_LABELS;
const SORT_OPTIONS = Object.entries(SORT_LABELS).map(([value, label]) => ({ value, label }));

function isSortOption(value: string): value is SortOption {
  return value in SORT_LABELS;
}

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
      <Select
        value={sortBy}
        items={SORT_OPTIONS}
        onValueChange={(v) => {
          if (v !== null && isSortOption(v)) {
            onSortChange(v);
          }
        }}
      >
        <SelectTrigger
          wrapperClassName="md:w-46"
          className="h-8 w-full bg-gray-1"
          leftIcon={<IconBarsFilterOutline18 className="size-3.5 text-gray-9" />}
          rightIcon={<IconChevronDownOutline18 className="size-3.5 absolute right-2" />}
        >
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          {SORT_OPTIONS.map((option) => (
            <SelectItem key={option.value} value={option.value}>
              {option.label}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </ResourceListHeader>
  );
}
