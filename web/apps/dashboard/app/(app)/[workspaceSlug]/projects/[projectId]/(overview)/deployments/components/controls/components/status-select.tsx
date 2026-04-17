"use client";

import { ChevronDown } from "@unkey/icons";
import { Checkbox, Popover, PopoverContent, PopoverTrigger } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import {
  DEPLOYMENT_STATUS_META,
  GROUPED_DEPLOYMENT_STATUSES,
  type GroupedDeploymentStatus,
} from "../../../filters.schema";
import { useFilters } from "../../../hooks/use-filters";

const STATUS_OPTIONS = GROUPED_DEPLOYMENT_STATUSES.map((value) => ({
  value,
  ...DEPLOYMENT_STATUS_META[value],
}));

export function StatusSelect() {
  const { filters, updateFilters } = useFilters();

  const selectedStatuses = filters.flatMap((f) =>
    f.field === "status" && typeof f.value === "string" ? [f.value] : [],
  );

  const toggleStatus = (status: GroupedDeploymentStatus) => {
    const isSelected = selectedStatuses.includes(status);
    const otherFilters = filters.filter((f) => f.field !== "status");
    const currentStatusFilters = filters.filter((f) => f.field === "status");

    if (isSelected) {
      const remaining = currentStatusFilters.filter((f) => f.value !== status);
      updateFilters([...otherFilters, ...remaining]);
    } else {
      updateFilters([
        ...otherFilters,
        ...currentStatusFilters,
        {
          field: "status",
          id: crypto.randomUUID(),
          operator: "is",
          value: status,
        },
      ]);
    }
  };

  const count = selectedStatuses.length;

  return (
    <Popover>
      <PopoverTrigger asChild>
        <button
          type="button"
          className={cn(
            "flex items-center gap-2 h-9 px-3 w-full",
            "bg-gray-1 border border-grayA-4 rounded-lg",
            "text-[13px] text-accent-12 font-normal",
            "hover:bg-gray-2 transition-colors",
            count > 0 && "bg-gray-2",
          )}
        >
          <span className="truncate">
            Status
            {count > 0 && (
              <span className="ml-1.5 inline-flex items-center justify-center bg-gray-7 rounded-sm h-4 px-1 text-[11px] font-medium">
                {count}
              </span>
            )}
          </span>
          <ChevronDown className="ml-auto shrink-0" iconSize="md-medium" />
        </button>
      </PopoverTrigger>
      <PopoverContent align="start" className="w-48 p-1">
        {STATUS_OPTIONS.map((option) => (
          <button
            type="button"
            key={option.value}
            className="flex items-center gap-2 px-2 py-1.5 rounded-md hover:bg-gray-3 cursor-pointer text-[13px] w-full"
            onClick={() => toggleStatus(option.value)}
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
