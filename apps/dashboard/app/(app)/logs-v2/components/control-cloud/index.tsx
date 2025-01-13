import { KeyboardButton } from "@/components/keyboard-button";
import { cn } from "@/lib/utils";
import { XMark } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { useCallback } from "react";
import type { FilterValue } from "../../filters.type";
import { useFilters } from "../../hooks/use-filters";
import { useKeyboardShortcut } from "../../hooks/use-keyboard-shortcut";

const formatFieldName = (field: string): string => {
  switch (field) {
    case "status":
      return "Status";
    case "paths":
      return "Path";
    case "methods":
      return "Method";
    case "requestId":
      return "Request ID";
    default:
      // Capitalize first letter
      return field.charAt(0).toUpperCase() + field.slice(1);
  }
};

const formatValue = (value: string | number): string => {
  if (typeof value === "string" && /^\d+$/.test(value)) {
    const statusFamily = Math.floor(Number.parseInt(value) / 100);
    switch (statusFamily) {
      case 5:
        return "5XX (Error)";
      case 4:
        return "4XX (Warning)";
      case 2:
        return "2XX (Success)";
      default:
        return `${statusFamily}xx`;
    }
  }
  return String(value);
};

type ControlPillProps = {
  filter: FilterValue;
  onRemove: (id: string) => void;
};

const ControlPill = ({ filter, onRemove }: ControlPillProps) => {
  const { field, operator, value, metadata } = filter;

  return (
    <div className="flex gap-0.5 font-mono">
      <div className="bg-gray-3 px-2 rounded-l-md text-accent-12 font-medium py-[2px]">
        {formatFieldName(field)}
      </div>
      <div className="bg-gray-3 px-2 text-accent-12 font-medium py-[2px] flex gap-1 items-center">
        {operator}
      </div>
      <div className="bg-gray-3 px-2 text-accent-12 font-medium py-[2px] flex gap-1 items-center">
        {metadata?.colorClass && (
          <div className={cn("size-2 rounded-[2px]", metadata.colorClass)} />
        )}
        {metadata?.icon}
        <span className="text-accent-12 text-xs font-mono">
          {formatValue(value)}
        </span>
      </div>
      <Button
        onClick={() => onRemove(filter.id)}
        className="bg-gray-3 rounded-none rounded-r-md py-[2px] px-2 [&_svg]:stroke-[2px] [&_svg]:size-3 flex items-center border-none h-auto"
      >
        <XMark className="text-gray-9" />
      </Button>
    </div>
  );
};

export const ControlCloud = () => {
  const { filters, removeFilter, updateFilters } = useFilters();

  useKeyboardShortcut({ key: "d", meta: true }, () => {
    updateFilters([]);
  });

  const handleRemoveFilter = useCallback(
    (id: string) => {
      removeFilter(id);
    },
    [removeFilter]
  );

  if (filters.length === 0) {
    return null;
  }

  return (
    <div className="px-3 py-2 w-full flex items-center min-h-10 border-b border-gray-4 gap-2 text-xs flex-wrap">
      {filters.map((filter) => (
        <ControlPill
          key={filter.id}
          filter={filter}
          onRemove={handleRemoveFilter}
        />
      ))}
      <div className="flex items-center px-2 py-1 gap-1 ml-auto">
        <span className="text-gray-9 text-[13px]">Clear filters</span>
        <KeyboardButton shortcut="d" modifierKey="⌘" />
      </div>
    </div>
  );
};
