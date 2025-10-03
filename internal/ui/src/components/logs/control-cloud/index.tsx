"use client";
// biome-ignore lint: React in this context is used throughout, so biome will change to types because no APIs are used even though React is needed.
import * as React from "react";
import { useCallback, useState } from "react";
import { useKeyboardShortcut } from "../../../hooks/use-keyboard-shortcut";
import type { FilterValue } from "../../../validation/filter.types";
import { KeyboardButton } from "../../buttons/keyboard-button";
import { ControlPill } from "./control-pill";
import { defaultFormatValue } from "./utils";

type ControlCloudProps<TFilter extends FilterValue> = {
  filters: TFilter[];
  removeFilter: (id: string) => void;
  updateFilters: (filters: TFilter[]) => void;
  formatFieldName: (field: string) => string;
  formatValue?: (value: string | number, field: string) => string;
  historicalWindow?: number;
};

export const ControlCloud = <TFilter extends FilterValue>({
  filters,
  removeFilter,
  updateFilters,
  formatFieldName,
  formatValue = defaultFormatValue,
  historicalWindow = 12 * 60 * 60 * 1000,
}: ControlCloudProps<TFilter>) => {
  const [focusedIndex, setFocusedIndex] = useState<number | null>(null);

  useKeyboardShortcut("option+shift+a", () => {
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
        value: timestamp - historicalWindow,
        id: crypto.randomUUID(),
        operator: "is",
      },
    ] as TFilter[]);
  });

  useKeyboardShortcut("option+shift+s", () => {
    setFocusedIndex(0);
  });

  const handleRemoveFilter = useCallback(
    (id: string) => {
      removeFilter(id);
      if (focusedIndex !== null) {
        if (focusedIndex >= filters.length - 1) {
          setFocusedIndex(Math.max(filters.length - 2, 0));
        }
      }
    },
    [removeFilter, filters.length, focusedIndex],
  );

  const handleKeyDown = (e: React.KeyboardEvent<HTMLDivElement>) => {
    if (filters.length === 0) {
      return;
    }

    const findNextInDirection = (direction: "up" | "down") => {
      if (focusedIndex === null) {
        return 0;
      }

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
        const isAbove = direction === "up" && rect.bottom < currentRect.top;
        const isBelow = direction === "down" && rect.top > currentRect.bottom;

        if (isAbove || isBelow) {
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
      className="px-3 py-2 w-full flex items-center min-h-10 border-b border-gray-4 gap-2 text-xs flex-wrap group"
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
          formatFieldName={formatFieldName}
          formatValue={formatValue}
        />
      ))}
      <div className="flex items-center px-2 py-1 gap-2 ml-auto max-md:hidden">
        <div className="flex items-center gap-2">
          <span className="text-gray-9 text-[13px]">Clear filters</span>
          <div className="max-w-0 opacity-0 group-hover:max-w-[100px] group-hover:opacity-100 transition-all duration-300 ease-in-out overflow-hidden">
            <KeyboardButton shortcut="⌥+⇧+A" />
          </div>
        </div>
        <div className="w-px h-4 bg-gray-4 mr-2" />
        <div className="flex items-center gap-2">
          <span className="text-gray-9 text-[13px]">Focus filters</span>
          <div className="max-w-0 opacity-0 group-hover:max-w-[100px] group-hover:opacity-100 transition-all duration-300 ease-in-out overflow-hidden">
            <KeyboardButton shortcut="⌥+⇧+S" />
          </div>
        </div>
      </div>
    </div>
  );
};
