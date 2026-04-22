"use client";

import { cn } from "@/lib/utils";
import { Checkbox, Popover, PopoverContent, PopoverTrigger } from "@unkey/ui";
import { DEPLOYMENT_STATUS_META, GROUPED_DEPLOYMENT_STATUSES } from "../../../filters.schema";
import { useFilters } from "../../../hooks/use-filters";
import { FilterTriggerButton } from "./filter-trigger-button";

const STATUS_OPTIONS = GROUPED_DEPLOYMENT_STATUSES.map((value) => ({
  value,
  ...DEPLOYMENT_STATUS_META[value],
}));

export function StatusSelect() {
  const { filters, toggleArrayFilter } = useFilters();

  const selectedStatuses = filters.flatMap((f) =>
    f.field === "status" && typeof f.value === "string" ? [f.value] : [],
  );

  return (
    <Popover>
      <PopoverTrigger asChild>
        <FilterTriggerButton
          label="Status"
          count={selectedStatuses.length}
          isActive={selectedStatuses.length > 0}
        />
      </PopoverTrigger>
      <PopoverContent align="start" className="w-48 p-1">
        {STATUS_OPTIONS.map((option) => (
          <button
            type="button"
            key={option.value}
            className="flex items-center gap-2 px-2 py-1.5 rounded-md hover:bg-gray-3 cursor-pointer text-[13px] w-full"
            onClick={() => toggleArrayFilter("status", option.value)}
          >
            <Checkbox
              variant="primary"
              size="md"
              checked={selectedStatuses.includes(option.value)}
              tabIndex={-1}
            />
            <span className={cn("size-2 rounded-full shrink-0", option.colorClass)} />
            <span className="text-accent-12">{option.label}</span>
          </button>
        ))}
      </PopoverContent>
    </Popover>
  );
}
