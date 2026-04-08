"use client";

import { ChevronDown, CircleInfo } from "@unkey/icons";
import { InfoTooltip } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import type { ReactNode } from "react";

export function AccordionSection({
  label,
  summary,
  active,
  onToggle,
  children,
  tooltipContent,
  headerAction,
}: {
  label: string;
  summary: ReactNode;
  active: boolean;
  onToggle: () => void;
  children: ReactNode;
  tooltipContent?: ReactNode;
  headerAction?: ReactNode;
}) {
  return (
    <div className="border-t border-grayA-4">
      <div className="flex items-center hover:bg-grayA-2 transition-colors">
        <button
          type="button"
          onClick={onToggle}
          className="flex-1 min-w-0 px-8 py-4 flex items-center justify-between gap-4 cursor-pointer"
        >
          <span className="flex items-center gap-2 text-[13px] text-gray-11 font-medium">
            <ChevronDown
              iconSize="sm-regular"
              className={cn("transition-transform duration-200", active ? "" : "-rotate-90")}
            />
            {label}
            {tooltipContent && (
              <InfoTooltip content={tooltipContent} asChild>
                <span
                  className="ml-0.5 inline-flex items-center text-gray-9 hover:text-gray-11"
                  onClick={(e) => e.stopPropagation()}
                >
                  <CircleInfo iconSize="md-medium" aria-hidden="true" />
                  <span className="sr-only">More info</span>
                </span>
              </InfoTooltip>
            )}
          </span>
          <span className="text-[12px] text-gray-11 truncate">{summary}</span>
        </button>
        {headerAction && <div className="pr-8 shrink-0">{headerAction}</div>}
      </div>
      {active && <div className="px-8 pb-6 pt-3">{children}</div>}
    </div>
  );
}
