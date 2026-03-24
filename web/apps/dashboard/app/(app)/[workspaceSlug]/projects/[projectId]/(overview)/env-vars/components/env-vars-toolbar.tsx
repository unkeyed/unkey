"use client";

import { BarsFilter, ChevronDown, InputSearch, Magnifier } from "@unkey/icons";
import { FormInput, Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@unkey/ui";

const SORT_OPTIONS = ["last-updated", "name-asc"] as const;
export type SortOption = (typeof SORT_OPTIONS)[number];
export type EnvironmentFilter = "all" | string;

function isSortOption(value: string): value is SortOption {
  return (SORT_OPTIONS as readonly string[]).includes(value);
}

type EnvVarsToolbarProps = {
  searchQuery: string;
  onSearchChange: (value: string) => void;
  environmentFilter: EnvironmentFilter;
  onEnvironmentFilterChange: (value: EnvironmentFilter) => void;
  environments: { id: string; slug: string }[];
  sortBy: SortOption;
  onSortChange: (value: SortOption) => void;
};

export function EnvVarsToolbar({
  searchQuery,
  onSearchChange,
  environmentFilter,
  onEnvironmentFilterChange,
  environments,
  sortBy,
  onSortChange,
}: EnvVarsToolbarProps) {
  return (
    <div className="flex flex-col md:flex-row items-stretch gap-2">
      <div className="flex-[50%]">
        <FormInput
          placeholder="Search..."
          value={searchQuery}
          onChange={(e) => onSearchChange(e.target.value)}
          className="[&_input]:h-9 [&_input]:text-[13px] w-full bg-gray-1 [&_input]:bg-gray-1"
          leftIcon={<Magnifier iconSize="lg-medium" className="text-gray-9" />}
        />
      </div>
      <div className="flex-[25%]">
        <Select value={environmentFilter} onValueChange={onEnvironmentFilterChange}>
          <SelectTrigger className="h-9 w-full bg-gray-1" rightIcon={<ChevronDown className="absolute right-2" iconSize="md-medium" />}>
            <SelectValue placeholder="All Environments" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All Environments</SelectItem>
            {environments.map((env) => (
              <SelectItem key={env.id} value={env.id} className="capitalize">
                {env.slug}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
      <div className="flex-[25%] max-w-[184px]">
        <Select value={sortBy} onValueChange={(v) => { if (isSortOption(v)) { onSortChange(v); } }}>
          <SelectTrigger
            className="h-9 w-full bg-gray-1"
            leftIcon={<BarsFilter iconSize="md-medium" className="text-gray-9" />}
            rightIcon={<ChevronDown className="absolute right-2" iconSize="md-medium" />}
          >
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="last-updated">Last Updated</SelectItem>
            <SelectItem value="name-asc">Name A-Z</SelectItem>
          </SelectContent>
        </Select>
      </div>
    </div >
  );
}
