"use client";
import { format, formatDistanceToNow, fromUnixTime } from "date-fns";
// biome-ignore lint: React in this context is used throughout, so biome will change to types because no APIs are used even though React is needed.
import * as React from "react";
import { useEffect, useRef, useState } from "react";
import { cn } from "../lib/utils";
import { Tooltip, TooltipContent, TooltipTrigger } from "./tooltip";

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

type DisplayType = "local" | "utc" | "relative";

const TimestampInfo: React.FC<{
  value: string | number;
  className?: string;
  displayType?: DisplayType;
  triggerRef?: React.RefObject<HTMLElement>;
  open?: boolean;
  onOpenChange?: (open: boolean) => void;
}> = ({
  value,
  className,
  displayType = "local",
  triggerRef: externalTriggerRef,
  open,
  onOpenChange,
}: {
  className?: string;
  value: string | number;
  displayType?: DisplayType;
  triggerRef?: React.RefObject<HTMLElement>;
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
    <Tooltip open={isOpen} onOpenChange={setIsOpen}>
      {externalTriggerRef ? (
        // If external trigger is provided, use a span and the external trigger
        <>
          <TooltipTrigger asChild>
            <span className={cn("text-xs", className)}>{getDisplayValue()}</span>
          </TooltipTrigger>
        </>
      ) : (
        // Otherwise use the internal trigger ref for the button
        <TooltipTrigger ref={internalTriggerRef} className={cn("text-xs", className)}>
          <span>{getDisplayValue()}</span>
        </TooltipTrigger>
      )}
      <TooltipContent
        align={align}
        side="right"
        className="font-mono p-0 bg-gray-1 shadow-2xl text-xs border rounded-lg w-auto min-w-[280px] z-50 overflow-hidden border-grayA-4"
      >
        <div className="py-3">
          <TooltipRow label="UTC" value={utc} />
          <TooltipRow label={localTimezone} value={local} />
          <TooltipRow label="Relative" value={relative} />
          <TooltipRow label="Timestamp" value={String(value)} />
        </div>
      </TooltipContent>
    </Tooltip>
  );
};
TimestampInfo.displayName = "TimestampInfo";
export { TimestampInfo };
