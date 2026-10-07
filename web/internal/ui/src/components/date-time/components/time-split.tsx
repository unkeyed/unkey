import { IconClockOutline18 } from "@unkey/icons";
import { isSameDay } from "date-fns";
import { useState } from "react";
// biome-ignore lint: React in this context is used throughout, so biome will change to types because no APIs are used even though React is needed.
import * as React from "react";
import type { DateRange } from "react-day-picker";
import { cn } from "../../../lib/utils";
import { useDateTimeContext } from "../date-time";

export type TimeUnit = {
  HH: string;
  mm: string;
  ss: string;
};

type TimeInputType = "start" | "end";
type TimeField = keyof TimeUnit;

type TimeSplitInputProps = {
  inputClassNames?: string;
  type: TimeInputType;
};
const MAX_VALUES = {
  HH: 23,
  mm: 59,
  ss: 59,
} as const;

export const compareTimeUnits = (time1: TimeUnit, time2: TimeUnit): number => {
  const t1 = Number(time1.HH) * 3600 + Number(time1.mm) * 60 + Number(time1.ss);
  const t2 = Number(time2.HH) * 3600 + Number(time2.mm) * 60 + Number(time2.ss);
  return t1 - t2;
};

export const isSingleDay = (range?: DateRange) =>
  !range?.from || !range.to || isSameDay(range.from, range.to);

const TimeSplitInput: React.FC<TimeSplitInputProps> = ({ type }) => {
  const { startTime, endTime, date, onTimeChange } = useDateTimeContext();
  const [focus, setFocus] = useState(false);
  const [draft, setDraft] = useState<TimeUnit | null>(null);
  const time = draft ?? (type === "start" ? startTime : endTime);

  const normalizeTimeUnit = (time: TimeUnit): TimeUnit => ({
    HH: time.HH.padStart(2, "0"),
    mm: time.mm.padStart(2, "0"),
    ss: time.ss.padStart(2, "0"),
  });

  const handleBlur = () => {
    const normalizedTime = normalizeTimeUnit(time);
    setDraft(null);
    setFocus(false);
    if (compareTimeUnits(normalizedTime, type === "start" ? startTime : endTime) === 0) {
      return;
    }

    const resolveConflicts = isSingleDay(date);
    if (type === "start") {
      const pushEnd = resolveConflicts && compareTimeUnits(normalizedTime, endTime) > 0;
      onTimeChange(normalizedTime, pushEnd ? normalizedTime : endTime);
    } else {
      const pullStart = resolveConflicts && compareTimeUnits(normalizedTime, startTime) < 0;
      onTimeChange(pullStart ? normalizedTime : startTime, normalizedTime);
    }
  };

  const handleChange = (value: string, field: TimeField) => {
    if (!/^\d{0,2}$/.test(value) || Number(value) > MAX_VALUES[field]) {
      return;
    }
    setDraft({ ...time, [field]: value });
  };

  const handleFocus = (event: React.FocusEvent<HTMLInputElement>) => {
    event.target.select();
    setFocus(true);
  };

  const inputClassNames = `
    w-5
    bg-transparent
    outline-hidden ring-0 focus:ring-0
    text-center
    text-gray-12 leading-6 tracking-normal font-medium text-sm
  `;

  const renderTimeInput = (field: TimeField, ariaLabel: string) => (
    <input
      type="text"
      value={time[field]}
      onChange={(e) => handleChange(e.target.value, field)}
      onBlur={handleBlur}
      onFocus={handleFocus}
      placeholder="00"
      aria-label={ariaLabel}
      className={inputClassNames}
    />
  );

  return (
    <div
      className={cn(
        "flex h-8 w-full items-center rounded-sm border  bg-raised text-gray-12",
        focus && " border-gray-10",
      )}
    >
      <IconClockOutline18 className="size-3.5 text-gray-9 m-3" />
      {renderTimeInput("HH", "Hours")}
      <span className="text-gray-12 leading-6 tracking-normal font-medium text-sm">:</span>
      {renderTimeInput("mm", "Minutes")}
      <span className="text-gray-12 leading-6 font-medium text-sm">:</span>
      {renderTimeInput("ss", "Seconds")}
      <span className="text-gray-12 leading-6 font-medium text-sm"> </span>
      {/* AM/PM and timezone still needs to be implemented */}
      {/* {renderTimeInput("")} */}
    </div>
  );
};
type TimeInputProps = {
  type: "range" | "single";
  className?: string;
};
export const TimeInput: React.FC<TimeInputProps> = ({ type, className }) => {
  return (
    <div
      className={cn(
        "w-full h-full flex flex-row items-center justify-center gap-2 mt-1",
        className,
      )}
    >
      <TimeSplitInput type="start" />
      {type === "range" ? <TimeSplitInput type="end" /> : null}
    </div>
  );
};
