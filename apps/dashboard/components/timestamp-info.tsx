//https://github.com/supabase/supabase/blob/master/packages/ui-patterns/TimestampInfo/index.tsx
"use client";

import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";
import { format, formatRelative, fromUnixTime } from "date-fns";
import { useEffect, useRef, useState } from "react";

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
  return format(date, "dd MMM HH:mm:ss");
};

const timestampUtcFormatter = (value: string | number) => {
  const date = isUnixMicro(value) ? unixMicroToDate(value) : new Date(value);
  const isoDate = date.toISOString();
  const utcDate = `${isoDate.substring(0, 10)} ${isoDate.substring(11, 19)}`;
  return format(utcDate, "dd MMM HH:mm:ss");
};

const timestampRelativeFormatter = (value: string | number) => {
  const date = isUnixMicro(value) ? unixMicroToDate(value) : new Date(value);
  return formatRelative(date, new Date());
};

export const TimestampInfo = ({
  value,
  className,
}: {
  className?: string;
  value: string | number;
}) => {
  const local = timestampLocalFormatter(value);
  const utc = timestampUtcFormatter(value);
  const relative = timestampRelativeFormatter(value);
  const [align, setAlign] = useState<"start" | "end">("start");
  const triggerRef = useRef<HTMLButtonElement>(null);
  const localTimezone = Intl.DateTimeFormat().resolvedOptions().timeZone;

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
  }, []);

  const TooltipRow = ({ label, value }: { label: string; value: string }) => {
    const [copied, setCopied] = useState(false);
    return (
      // biome-ignore lint/a11y/useKeyWithClickEvents: <explanation>
      <span
        onClick={(e) => {
          e.stopPropagation();
          navigator.clipboard.writeText(value);
          setCopied(true);
          setTimeout(() => {
            setCopied(false);
          }, 1000);
        }}
        className={cn(
          "flex items-center space-x-2 hover:bg-background-subtle text-left px-3 py-2",
          {
            "bg-background-subtle": copied,
          },
        )}
      >
        <span className="text-right truncate">{label}:</span>
        <span className={copied ? "text-success" : "capitalize"}>{copied ? "Copied!" : value}</span>
      </span>
    );
  };

  return (
    <Tooltip>
      <TooltipTrigger ref={triggerRef} className={cn("text-xs", className)}>
        <span>{timestampLocalFormatter(value)}</span>
      </TooltipTrigger>
      <TooltipContent
        align={align}
        side="right"
        className="font-mono p-0 bg-background shadow-md text-xs"
      >
        <TooltipRow label="UTC" value={utc} />
        <div className="border-b border-border" />
        <TooltipRow label={`${localTimezone}`} value={local} />
        <div className="border-b border-border" />
        <TooltipRow label="Relative" value={relative} />
        <div className="border-b border-border" />
        <TooltipRow label="Timestamp" value={String(value)} />
      </TooltipContent>
    </Tooltip>
  );
};
