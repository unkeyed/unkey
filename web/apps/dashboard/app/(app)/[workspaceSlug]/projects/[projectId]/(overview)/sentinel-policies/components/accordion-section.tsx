"use client";

import { ChevronDown, CircleInfo } from "@unkey/icons";
import { InfoTooltip } from "@unkey/ui";
import type { ReactNode } from "react";

export function AccordionSection({
  label,
  summary,
  active,
  onToggle,
  children,
  tooltipContent,
}: {
  label: string;
  summary: ReactNode;
  active: boolean;
  onToggle: () => void;
  children: ReactNode;
  tooltipContent?: ReactNode;
}) {
  return (
    <div className="border-t border-grayA-4">
      <button
        type="button"
        onClick={onToggle}
        className="w-full px-8 py-4 flex items-center justify-between gap-4 hover:bg-grayA-2 transition-colors cursor-pointer"
      >
        <span className="flex items-center gap-2 text-[13px] text-gray-11 font-medium">
          <ChevronDown
            iconSize="sm-regular"
            className={`transition-transform duration-200 ${active ? "" : "-rotate-90"}`}
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
      {active && <div className="px-8 pb-6 pt-3">{children}</div>}
    </div>
  );
}
