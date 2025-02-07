import { KeyboardButton } from "@/components/keyboard-button";
import { TimestampInfo } from "@/components/timestamp-info";
import { useKeyboardShortcut } from "@/hooks/use-keyboard-shortcut";
import { cn } from "@/lib/utils";
import { XMark } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { format } from "date-fns";
import { type KeyboardEvent, useCallback, useEffect, useRef, useState } from "react";
import type { FilterValue } from "../../filters.type";
import { useFilters } from "../../hooks/use-filters";
import { HISTORICAL_DATA_WINDOW } from "../table/hooks/use-logs-query";

const formatFieldName = (field: string): string => {
  switch (field) {
    case "startTime":
      return "Start time";
    case "endTime":
      return "End time";
    case "status":
      return "Status";
    case "requestId":
      return "Request ID";
    case "identifiers":
      return "Identifier";
    case "since":
      return "";
    default:
      return field.charAt(0).toUpperCase() + field.slice(1);
  }
};

const formatValue = (value: string | number, field: string): string => {
  if (typeof value === "number" && (field === "startTime" || field === "endTime")) {
    return format(value, "MMM d, yyyy HH:mm:ss");
  }
  return String(value);
};

const formatOperator = (operator: string, field: string): string => {
  if (field === "since" && operator === "is") {
    return "Last";
  }
  return operator;
};

type ControlPillProps = {
  filter: FilterValue;
  onRemove: (id: string) => void;
  isFocused?: boolean;
  onFocus?: () => void;
  index: number;
};

const ControlPill = ({ filter, onRemove, isFocused, onFocus, index }: ControlPillProps) => {
  const { field, operator, value, metadata } = filter;
  const pillRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (isFocused && pillRef.current) {
      const button = pillRef.current.querySelector("button");
      button?.focus();
    }
  }, [isFocused]);

  const handleKeyDown = (e: KeyboardEvent) => {
    if (e.key === "Backspace" || e.key === "Delete") {
      e.preventDefault();
      onRemove(filter.id);
    }
  };

  return (
    <div className="flex gap-0.5 font-mono group" data-pill-index={index}>
      {formatFieldName(field) === "" ? null : (
        <div className="bg-gray-3 px-2 rounded-l-md text-accent-12 font-medium py-[2px]">
          {formatFieldName(field)}
        </div>
      )}
      <div className="bg-gray-3 px-2 text-accent-12 font-medium py-[2px] flex gap-1 items-center">
        {formatOperator(operator, field)}
      </div>
      <div className="bg-gray-3 px-2 text-accent-12 font-medium py-[2px] flex gap-1 items-center">
        {metadata?.colorClass && (
          <div className={cn("size-2 rounded-[2px]", metadata.colorClass)} />
        )}
        {field === "endTime" || field === "startTime" ? (
          <TimestampInfo
            value={formatValue(value, field)}
            className={cn("font-mono group-hover:underline decoration-dotted")}
          />
        ) : (
          <span className="text-accent-12 text-xs font-mono">{formatValue(value, field)}</span>
        )}
      </div>
      <div ref={pillRef} className="contents">
        <Button
          onClick={() => onRemove(filter.id)}
          onFocus={onFocus}
          onKeyDown={handleKeyDown}
          tabIndex={0}
          className={cn(
            "bg-gray-3 rounded-none rounded-r-md py-[2px] px-2 [&_svg]:stroke-[2px] [&_svg]:size-3 flex items-center border-none h-auto focus:ring-2 focus:ring-accent-7 focus:outline-none",
            isFocused && "bg-gray-4",
          )}
        >
          <XMark className={cn("text-gray-9", isFocused && "text-gray-11")} />
        </Button>
      </div>
    </div>
  );
};

export const RatelimitLogsControlCloud = () => {
  const { filters, removeFilter, updateFilters } = useFilters();
  const [focusedIndex, setFocusedIndex] = useState<number | null>(null);

  useKeyboardShortcut({ key: "d", meta: true }, () => {
    const timestamp = Date.now();
    updateFilters([
      {
        field: "endTime",
        value: timestamp,
        id: crypto.randomUUID(),
        operator: "is",
      },
      {
        field: "startTime",
        value: timestamp - HISTORICAL_DATA_WINDOW,
        id: crypto.randomUUID(),
        operator: "is",
      },
    ]);
  });

  useKeyboardShortcut({ key: "c", meta: true }, () => {
    setFocusedIndex(0);
  });

  const handleRemoveFilter = useCallback(
    (id: string) => {
      removeFilter(id);
      // Adjust focus after removal
      if (focusedIndex !== null) {
        if (focusedIndex >= filters.length - 1) {
          setFocusedIndex(Math.max(filters.length - 2, 0));
        }
      }
    },
    [removeFilter, filters.length, focusedIndex],
  );

  const handleKeyDown = (e: KeyboardEvent) => {
    if (filters.length === 0) {
      return;
    }

    const findNextInDirection = (direction: "up" | "down") => {
      if (focusedIndex === null) {
        return 0;
      }

      // Get all buttons
      const buttons = document.querySelectorAll("[data-pill-index] button");
      const currentButton = buttons[focusedIndex] as HTMLElement;
      if (!currentButton) {
        return focusedIndex;
      }

      const currentRect = currentButton.getBoundingClientRect();
      let closestDistance = Number.POSITIVE_INFINITY;
      let closestIndex = focusedIndex;

      buttons.forEach((button, index) => {
        const rect = button.getBoundingClientRect();

        // Check if item is in the row above/below
        const isAbove = direction === "up" && rect.bottom < currentRect.top;
        const isBelow = direction === "down" && rect.top > currentRect.bottom;

        if (isAbove || isBelow) {
          // Calculate horizontal distance
          const horizontalDistance = Math.abs(rect.left - currentRect.left);
          if (horizontalDistance < closestDistance) {
            closestDistance = horizontalDistance;
            closestIndex = index;
          }
        }
      });

      return closestIndex;
    };

    switch (e.key) {
      case "ArrowRight":
      case "l":
        e.preventDefault();
        setFocusedIndex((prev) => (prev === null ? 0 : (prev + 1) % filters.length));
        break;
      case "ArrowLeft":
      case "h":
        e.preventDefault();
        setFocusedIndex((prev) =>
          prev === null ? filters.length - 1 : (prev - 1 + filters.length) % filters.length,
        );
        break;
      case "ArrowDown":
      case "j":
        e.preventDefault();
        setFocusedIndex(findNextInDirection("down"));
        break;
      case "ArrowUp":
      case "k":
        e.preventDefault();
        setFocusedIndex(findNextInDirection("up"));
        break;
    }
  };

  if (filters.length === 0) {
    return null;
  }

  return (
    <div
      className="px-3 py-2 w-full flex items-center min-h-10 border-b border-gray-4 gap-2 text-xs flex-wrap"
      onKeyDown={handleKeyDown}
    >
      {filters.map((filter, index) => (
        <ControlPill
          key={filter.id}
          filter={filter}
          onRemove={handleRemoveFilter}
          isFocused={focusedIndex === index}
          onFocus={() => setFocusedIndex(index)}
          index={index}
        />
      ))}
      <div className="flex items-center px-2 py-1 gap-2 ml-auto">
        <span className="text-gray-9 text-[13px]">Clear filters</span>
        <KeyboardButton shortcut="d" modifierKey="⌘" />
        <div className="w-px h-4 bg-gray-4" />
        <span className="text-gray-9 text-[13px]">Focus filters</span>
        <KeyboardButton shortcut="c" modifierKey="⌘" />
      </div>
    </div>
  );
};
