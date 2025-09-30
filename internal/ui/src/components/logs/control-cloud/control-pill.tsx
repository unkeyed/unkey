// biome-ignore lint: React in this context is used throughout, so biome will change to types because no APIs are used even though React is needed.
import * as React from "react";
import { cn } from "../../../lib/utils";
import { XMark } from "@unkey/icons";
import { TimestampInfo } from "../../timestamp-info";
import { Button } from "../../buttons/button";
import type { FilterValue } from "../../../validation/filter.types";
import { formatOperator } from "./utils";
import { useEffect, useRef } from "react";

type ControlPillProps<T extends FilterValue> = {
  filter: T;
  onRemove: (id: string) => void;
  isFocused?: boolean;
  onFocus?: () => void;
  index: number;
  formatFieldName: (field: string) => string;
  formatValue: (value: string | number, field: string) => string;
};

export const ControlPill = <TFilter extends FilterValue>({
  filter,
  onRemove,
  isFocused,
  onFocus,
  index,
  formatFieldName,
  formatValue,
}: ControlPillProps<TFilter>) => {
  const { field, operator, value, metadata } = filter;
  const pillRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (isFocused && pillRef.current) {
      const button = pillRef.current.querySelector("button");
      button?.focus();
    }
  }, [isFocused]);

  const handleKeyDown = (e: React.KeyboardEvent<HTMLButtonElement>) => {
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
      <div
        className={cn(
          "bg-gray-3 px-2 text-accent-12 font-medium py-[2px] flex gap-1 items-center",
          formatFieldName(field) === "" ? "rounded-l-md" : "",
        )}
      >
        {formatOperator(operator, field)}
      </div>
      <div className="bg-gray-3 px-2 text-accent-12 font-medium py-[2px] flex gap-1 items-center">
        {metadata?.colorClass && (
          <div className={cn("size-2 rounded-[2px]", metadata.colorClass)} />
        )}
        {metadata?.icon}
        {field === "endTime" || field === "startTime" ? (
          <TimestampInfo
            value={value}
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
            "bg-gray-3 rounded-none rounded-r-md py-[2px] px-2 [&_svg]:stroke-[2px] [&_svg]:size-3 flex items-center border-none h-auto focus:ring-2 focus:ring-offset-1 focus:ring-accent-9 focus:outline-none hover:bg-gray-4 focus:hover:bg-gray-4",
            isFocused && "bg-gray-4",
          )}
        >
          <XMark className={cn("text-gray-9", isFocused && "text-gray-11")} />
        </Button>
      </div>
    </div>
  );
};
