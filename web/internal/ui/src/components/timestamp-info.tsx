"use client";
import { format, formatDistanceToNow, fromUnixTime } from "date-fns";
import * as React from "react";
import { useEffect, useRef, useState } from "react";
import { cn } from "../lib/utils";
import { Popover, PopoverContent, PopoverTrigger } from "./dialog/popover";

const unixMicroToDate = (unix: string | number): Date => {
  return fromUnixTime(Number(unix) / 1000 / 1000);
};

const isUnixMicro = (unix: string | number): boolean => {
  const digitLength = String(unix).length === 16;
  const isNum = !Number.isNaN(Number(unix));
  return isNum && digitLength;
};

const timestampLocalFormatter = (value: string | number) => {
  const date = isUnixMicro(value) ? unixMicroToDate(value) : new Date(value);
  return format(date, "MMM dd HH:mm:ss");
};

const timestampLocalHoursWithMillisFormatter = (value: string | number) => {
  const date = isUnixMicro(value) ? unixMicroToDate(value) : new Date(value);
  return format(date, "HH:mm:ss.SSS");
};

const timestampUtcFormatter = (value: string | number) => {
  const date = isUnixMicro(value) ? unixMicroToDate(value) : new Date(value);
  const isoDate = date.toISOString();
  const utcDate = `${isoDate.substring(0, 10)} ${isoDate.substring(11, 19)}`;
  return format(utcDate, "MMM d,yyyy HH:mm:ss");
};

const timestampRelativeFormatter = (value: string | number): string => {
  const date = isUnixMicro(value) ? unixMicroToDate(value) : new Date(value);

  return formatDistanceToNow(date, {
    addSuffix: true,
  });
};

type DisplayType = "local" | "local_hours_with_millis" | "utc" | "relative";

const TimestampInfo: React.FC<{
  value: string | number;
  className?: string;
  displayType?: DisplayType;
  side?: "left" | "right" | "top" | "bottom";
  align?: "start" | "center" | "end";
  triggerRef?: React.RefObject<HTMLElement | null>;
  open?: boolean;
  onOpenChange?: (open: boolean) => void;
}> = ({
  value,
  className,
  displayType = "local",
  side: sideProp,
  align: alignProp,
  triggerRef: externalTriggerRef,
  open,
  onOpenChange,
}: {
  className?: string;
  value: string | number;
  displayType?: DisplayType;
  side?: "left" | "right" | "top" | "bottom";
  align?: "start" | "center" | "end";
  triggerRef?: React.RefObject<HTMLElement | null>;
  open?: boolean;
  onOpenChange?: (open: boolean) => void;
}) => {
  const local = timestampLocalFormatter(value);
  const utc = timestampUtcFormatter(value);
  const relative = timestampRelativeFormatter(value);
  const [align, setAlign] = useState<"start" | "end">("start");
  const internalTriggerRef = useRef<HTMLButtonElement>(null);
  const triggerRef = externalTriggerRef || internalTriggerRef;
  const localTimezone = Intl.DateTimeFormat().resolvedOptions().timeZone;
  const [internalOpen, setInternalOpen] = useState(false);

  // Use either controlled open state or internal state
  const isOpen = open !== undefined ? open : internalOpen;
  const setIsOpen = onOpenChange || setInternalOpen;

  // When an external trigger element is provided, the popup is anchored to it
  // instead of to our own trigger. The positioner expects a non-null current,
  // so fall back to the body (same pattern as ConfirmPopover).
  const externalAnchor = React.useMemo(() => {
    if (!externalTriggerRef) {
      return undefined;
    }
    return {
      get current() {
        return externalTriggerRef.current ?? document.body;
      },
    };
  }, [externalTriggerRef]);

  useEffect(() => {
    const updateAlignment = () => {
      if (triggerRef.current) {
        const rect = triggerRef.current.getBoundingClientRect();
        const windowHeight = window.innerHeight;
        setAlign(rect.top < windowHeight / 2 ? "start" : "end");
      }
    };
    updateAlignment();
    window.addEventListener("scroll", updateAlignment);
    window.addEventListener("resize", updateAlignment);
    return () => {
      window.removeEventListener("scroll", updateAlignment);
      window.removeEventListener("resize", updateAlignment);
    };
  }, [triggerRef]);

  const getDisplayValue = () => {
    switch (displayType) {
      case "local":
        return timestampLocalFormatter(value);
      case "utc":
        return utc;
      case "relative":
        return relative;
      case "local_hours_with_millis":
        return timestampLocalHoursWithMillisFormatter(value);
      default:
        return timestampLocalFormatter(value);
    }
  };

  const TooltipRow = ({ label, value }: { label: string; value: string }) => {
    const [copied, setCopied] = useState(false);
    return (
      //biome-ignore lint/a11y/useKeyWithClickEvents: no need
      <span
        onClick={(e) => {
          e.stopPropagation();
          navigator.clipboard.writeText(value);
          setCopied(true);
          setTimeout(() => setCopied(false), 1000);
        }}
        className="flex items-center hover:bg-gray-3 text-left cursor-pointer w-full px-5 py-2"
      >
        <span className="w-32 text-left truncate text-accent-9">{label}</span>
        <span className={cn("ml-2 text-xs text-accent-12", copied ? "text-success-11" : "")}>
          {copied ? "Copied!" : value}
        </span>
      </span>
    );
  };

  return (
    <Popover open={isOpen} onOpenChange={setIsOpen}>
      {externalTriggerRef ? (
        // An external element controls the popover (via open/onOpenChange) and
        // anchors it, so only render the timestamp text here.
        <span className={cn("text-xs", className)}>{getDisplayValue()}</span>
      ) : (
        // A real, focusable trigger: the popup contains interactive copy rows,
        // so it must be reachable by keyboard. `openOnHover` keeps the old
        // tooltip-like hover behavior; click/Enter/Space also toggle it.
        <PopoverTrigger ref={internalTriggerRef} openOnHover className={cn("text-xs", className)}>
          {getDisplayValue()}
        </PopoverTrigger>
      )}
      <PopoverContent
        align={alignProp ?? align}
        side={sideProp ?? "right"}
        anchor={externalAnchor}
        className="font-mono p-0 bg-gray-1 shadow-2xl text-xs border rounded-lg w-auto min-w-[280px] z-50 overflow-hidden border-grayA-4"
      >
        <div className="py-3">
          <TooltipRow label="UTC" value={utc} />
          <TooltipRow label={localTimezone} value={local} />
          <TooltipRow label="Relative" value={relative} />
          <TooltipRow label="Timestamp" value={String(value)} />
        </div>
      </PopoverContent>
    </Popover>
  );
};
TimestampInfo.displayName = "TimestampInfo";
export { TimestampInfo };
