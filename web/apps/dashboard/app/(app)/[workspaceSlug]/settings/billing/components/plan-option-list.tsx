"use client";

import { formatDollars } from "@/lib/fmt";
import { cn } from "cn";
import { useId } from "react";

export type PlanOption = {
  id: string;
  name: string;
  monthlyCents: number;
};

type PlanOptionListProps = {
  options: PlanOption[];
  currentId: string | null;
  selectedId: string | null;
  onSelect: (id: string) => void;
  label: string;
};

export function PlanOptionList({
  options,
  currentId,
  selectedId,
  onSelect,
  label,
}: PlanOptionListProps) {
  const name = useId();
  return (
    <fieldset
      aria-label={label}
      className="flex flex-col divide-y overflow-hidden rounded-xl border bg-raised"
    >
      {options.map((option) => {
        const isCurrent = option.id === currentId;
        const isSelected = option.id === selectedId;
        return (
          <label
            key={option.id}
            className={cn(
              "flex cursor-pointer items-center gap-3 px-4 py-3 transition-colors duration-150 ease-out",
              "has-[:focus-visible]:outline has-[:focus-visible]:outline-2 has-[:focus-visible]:-outline-offset-2 has-[:focus-visible]:outline-gray-8",
              isSelected ? "bg-grayA-2" : "hover:bg-grayA-2",
            )}
          >
            <input
              type="radio"
              name={name}
              value={option.id}
              checked={isSelected}
              onChange={() => onSelect(option.id)}
              className="sr-only"
            />
            <span
              aria-hidden="true"
              className={cn(
                "flex size-4 shrink-0 items-center justify-center rounded-full border transition-colors duration-150 ease-out",
                isSelected ? "border-gray-12 bg-gray-12" : "border-grayA-6",
              )}
            >
              <span
                className={cn(
                  "size-1.5 rounded-full bg-gray-1 transition-transform duration-150 ease-out",
                  isSelected ? "scale-100" : "scale-0",
                )}
              />
            </span>
            <span className="flex min-w-0 flex-1 items-center gap-2">
              <span className="shrink-0 font-medium text-gray-12 text-sm tabular-nums">
                {option.name}
              </span>
              {isCurrent ? (
                <span className="shrink-0 rounded-full bg-grayA-3 px-2 text-2xs text-gray-11 leading-4">
                  Current
                </span>
              ) : null}
            </span>
            <span className="shrink-0 text-gray-12 text-sm tabular-nums">
              {formatDollars(option.monthlyCents)}
              <span className="text-gray-11 text-xs">/mo</span>
            </span>
          </label>
        );
      })}
    </fieldset>
  );
}
