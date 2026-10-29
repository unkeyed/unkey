// biome-ignore lint: React in this context is used throughout, so biome will change to types because no APIs are used even though React is needed.
import * as React from "react";
import { cn } from "../../../lib/utils";
import { IconXmarkOutline18 } from "@unkey/icons";
import { TimestampInfo } from "../../timestamp-info";
import { Button } from "../../buttons/button";
import type { FilterValue } from "../../../validation/filter.types";
import { formatOperator } from "./utils";
import { useEffect, useRef } from "react";

type ControlPillVariant = "solid" | "outline";

type PillStyles = {
  root: string;
  field: string;
  operator: string;
  operatorWithoutField: string;
  value: string;
  remove: string;
  removeFocused: string;
};

const PILL_STYLES: Record<ControlPillVariant, PillStyles> = {
  solid: {
    root: "gap-0.5",
    field: "bg-gray-3 rounded-l-md text-gray-12",
    operator: "bg-gray-3 text-gray-12",
    operatorWithoutField: "rounded-l-md",
    value: "bg-gray-3",
    remove:
      "bg-gray-3 rounded-r-md hover:bg-gray-4 focus:hover:bg-gray-4 focus:ring-2 focus:ring-offset-1 focus:ring-gray-9",
    removeFocused: "bg-gray-4",
  },
  outline: {
    root: "divide-x divide-grayA-4 overflow-hidden rounded-md border border-grayA-4 bg-gray-1",
    field: "text-gray-11",
    operator: "text-gray-9",
    operatorWithoutField: "",
    value: "",
    remove:
      "hover:bg-grayA-3 focus:hover:bg-grayA-3 focus:ring-2 focus:ring-inset focus:ring-gray-9",
    removeFocused: "bg-grayA-3",
  },
};

type ControlPillProps<T extends FilterValue> = {
  variant?: ControlPillVariant;
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
  variant = "solid",
}: ControlPillProps<TFilter>) => {
  const styles = PILL_STYLES[variant];
  const { field, operator, value, metadata } = filter;
  const fieldName = formatFieldName(field);
  const operatorLabel = formatOperator(operator, field);
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
    <div className={cn("flex font-mono group", styles.root)} data-pill-index={index}>
      {fieldName === "" ? null : (
        <div className={cn("px-2 font-medium py-[2px]", styles.field)}>{fieldName}</div>
      )}
      <div
        className={cn(
          "px-2 font-medium py-[2px] flex gap-1 items-center",
          styles.operator,
          fieldName === "" && styles.operatorWithoutField,
        )}
      >
        {operatorLabel}
      </div>
      <div
        className={cn(
          "px-2 text-gray-12 font-medium py-[2px] flex gap-1 items-center",
          styles.value,
        )}
      >
        {metadata?.colorClass && <div className={cn("size-2 rounded-xs", metadata.colorClass)} />}
        {metadata?.icon}
        {field === "endTime" || field === "startTime" ? (
          <TimestampInfo
            value={value}
            className={cn("font-mono group-hover:underline decoration-dotted")}
          />
        ) : (
          <span className="text-gray-12 text-xs font-mono">{formatValue(value, field)}</span>
        )}
      </div>
      <div ref={pillRef} className="contents">
        <Button
          variant="ghost"
          onClick={() => onRemove(filter.id)}
          onFocus={onFocus}
          onKeyDown={handleKeyDown}
          tabIndex={0}
          aria-label={["Remove", fieldName, operatorLabel, formatValue(value, field)]
            .filter(Boolean)
            .join(" ")}
          className={cn(
            "rounded-none py-[2px] px-2 [&_svg]:stroke-[2px] [&_svg]:size-3 flex items-center border-none h-auto focus:outline-hidden",
            styles.remove,
            isFocused && styles.removeFocused,
          )}
        >
          <IconXmarkOutline18 className={cn("text-gray-9", isFocused && "text-gray-11")} />
        </Button>
      </div>
    </div>
  );
};
