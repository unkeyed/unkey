"use client";

import {
  IconBarsFilterOutline18,
  IconChevronDownOutline18,
  IconMagnifierOutline18,
} from "@unkey/icons";
import {
  InputGroup,
  InputGroupAddon,
  InputGroupInput,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@unkey/ui";

const SORT_OPTIONS = ["last-updated", "name-asc"] as const;
export type SortOption = (typeof SORT_OPTIONS)[number];

function isSortOption(value: string): value is SortOption {
  return (SORT_OPTIONS as readonly string[]).includes(value);
}

type EnvVarsToolbarProps = {
  searchQuery: string;
  onSearchChange: (value: string) => void;
  sortBy: SortOption;
  onSortChange: (value: SortOption) => void;
};

export function EnvVarsToolbar({
  searchQuery,
  onSearchChange,
  sortBy,
  onSortChange,
}: EnvVarsToolbarProps) {
  return (
    <div className="flex flex-col md:flex-row items-stretch gap-2">
      <div className="flex-1">
        <InputGroup className="h-9 w-full bg-gray-1">
          <InputGroupAddon className="pointer-events-none">
            <IconMagnifierOutline18 className="size-4 text-gray-9" />
          </InputGroupAddon>
          <InputGroupInput
            placeholder="Search..."
            value={searchQuery}
            onChange={(e) => onSearchChange(e.target.value)}
            className="h-9 text-sm"
          />
        </InputGroup>
      </div>
      <div className="w-full md:w-[184px]">
        <Select
          value={sortBy}
          items={[
            { value: "last-updated", label: "Last Updated" },
            { value: "name-asc", label: "Name A-Z" },
          ]}
          onValueChange={(v) => {
            if (v !== null && isSortOption(v)) {
              onSortChange(v);
            }
          }}
        >
          <SelectTrigger
            className="h-9 w-full bg-gray-1"
            leftIcon={<IconBarsFilterOutline18 className="size-3.5 text-gray-9" />}
            rightIcon={<IconChevronDownOutline18 className="size-3.5 absolute right-2" />}
          >
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="last-updated">Last Updated</SelectItem>
            <SelectItem value="name-asc">Name A-Z</SelectItem>
          </SelectContent>
        </Select>
      </div>
    </div>
  );
}
