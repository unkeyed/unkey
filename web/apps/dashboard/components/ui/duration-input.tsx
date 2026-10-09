"use client";

import { parseDuration } from "@/lib/duration";
import { formatMs } from "@/lib/ms";
import { FormInput } from "@unkey/ui";
import { type ComponentProps, useEffect, useRef, useState } from "react";

type DurationInputProps = Omit<
  ComponentProps<typeof FormInput>,
  "value" | "onChange" | "type" | "error"
> & {
  value: number;
  onChange: (ms: number) => void;
  error?: string;
};

const PARSE_ERROR = 'Use a duration like "5s", "2m", "1h" or milliseconds';

function toMs(raw: string): number | null {
  const trimmed = raw.trim();
  if (trimmed === "") {
    return 0;
  }
  const asNumber = Number(trimmed);
  if (Number.isFinite(asNumber) && asNumber > 0) {
    return Math.floor(asNumber);
  }
  const parsed = parseDuration(trimmed);
  return parsed > 0 ? parsed : null;
}

export function DurationInput({
  value,
  onChange,
  error,
  placeholder = "e.g. 5s, 2m, 1h, 500ms",
  ...props
}: DurationInputProps) {
  const [display, setDisplay] = useState(() => formatMs(value));
  const [parseError, setParseError] = useState<string>();
  const localValue = useRef(value);

  useEffect(() => {
    if (value !== localValue.current) {
      localValue.current = value;
      setDisplay(formatMs(value));
      setParseError(undefined);
    }
  }, [value]);

  return (
    <FormInput
      {...props}
      type="text"
      placeholder={placeholder}
      value={display}
      error={parseError ?? error}
      onChange={(e) => {
        setDisplay(e.target.value);
        const ms = toMs(e.target.value);
        if (ms === null) {
          setParseError(PARSE_ERROR);
          return;
        }
        setParseError(undefined);
        localValue.current = ms;
        onChange(ms);
      }}
    />
  );
}
