import { format } from "date-fns";

import { Clock } from "@unkey/icons";
import { useState } from "react";
// biome-ignore lint: React in this context is used throughout, so biome will change to types because no APIs are used even though React is needed.
import * as React from "react";
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

const TimeSplitInput: React.FC<TimeSplitInputProps> = ({ type }) => {
  const { startTime, endTime, date, onStartTimeChange, onEndTimeChange } = useDateTimeContext();
  const [focus, setFocus] = useState(false);
  const [time, setTime] = useState<TimeUnit>(type === "start" ? startTime : endTime);

  const normalizeTimeUnit = (time: TimeUnit): TimeUnit => ({
    HH: time.HH.padStart(2, "0") || "00",
    mm: time.mm.padStart(2, "0") || "00",
    ss: time.ss.padStart(2, "0") || "00",
  });

  const isSameDay = (date1: Date, date2: Date) =>
    format(new Date(date1), "dd/MM/yyyy") === format(new Date(date2), "dd/MM/yyyy");

  const compareTimeUnits = (time1: TimeUnit, time2: TimeUnit): number => {
    const t1 = Number(time1.HH) * 3600 + Number(time1.mm) * 60 + Number(time1.ss);
    const t2 = Number(time2.HH) * 3600 + Number(time2.mm) * 60 + Number(time2.ss);
    return t1 - t2;
  };

  const handleTimeConflicts = (normalizedTime: TimeUnit) => {
    // Only handle conflicts if start and end are on the same day
    if (date?.from && date.to) {
      if (!isSameDay(date.from, date?.to)) {
        return;
      }
    }
    // If this is a start time and it's later than the end time,
    // push the end time forward to match the start time
    if (type === "start" && endTime && compareTimeUnits(normalizedTime, endTime) > 0) {
      onEndTimeChange(normalizedTime);
    }
    // If this is an end time and it's earlier than the start time,
    // pull the start time backward to match the end time
    else if (type === "end" && startTime && compareTimeUnits(normalizedTime, startTime) < 0) {
      onStartTimeChange(normalizedTime);
    }
  };

  const updateTimeState = (normalizedTime: TimeUnit) => {
    setTime(normalizedTime);
    if (type === "start") {
      onStartTimeChange(normalizedTime);
    }
    if (type === "end") {
      onEndTimeChange(normalizedTime);
    }
  };

  const handleBlur = (value: string, field: TimeField) => {
    if (value !== time[field]) {
      handleChange(value, field);
    }
    const normalizedTime = normalizeTimeUnit(time);
    updateTimeState(normalizedTime);
    handleTimeConflicts(normalizedTime);
    setFocus(false);
  };

  const handleChange = (value: string, field: TimeField) => {
    if (value.length > 2) {
      return;
    }

    const numValue = Number(value);
    if (value && numValue > MAX_VALUES[field]) {
      return;
    }
    setTime({ ...time, [field]: value });
  };

  const handleFocus = (event: React.FocusEvent<HTMLInputElement>) => {
    event.target.select();
    setFocus(true);
  };

  const inputClassNames = `
    w-5
    bg-transparent
    outline-none ring-0 focus:ring-0
    text-center
    text-gray-12 leading-6 tracking-normal font-medium text-[13px]
  `;

  const TimeInput: React.FC<{ field: TimeField; ariaLabel: string }> = (props): JSX.Element => (
    <input
      type="text"
      value={time[props.field]}
      onChange={(e) => handleChange(e.target.value, props.field)}
      onBlur={(e) => handleBlur(e.target.value, props.field)}
      onFocus={handleFocus}
      placeholder="00"
      aria-label={props.ariaLabel}
      className={inputClassNames}
    />
  );

  return (
    <div
      className={cn(
        "flex h-8 w-full items-center rounded rounded-3 border-[1px]  bg-gray-2 text-gray-12",
        focus ? " border-gray-10" : "border-grayA-4",
      )}
    >
      <Clock className="text-gray-9 m-3 " />
      <TimeInput field="HH" ariaLabel="Hours" />
      <span className="text-gray-12 leading-6 tracking-normal font-medium text-[13px]">:</span>
      <TimeInput field="mm" ariaLabel="Minutes" />
      <span className="text-gray-12 leading-6 font-medium text-[13px]">:</span>
      <TimeInput field="ss" ariaLabel="Seconds" />
      <span className="text-gray-12 leading-6 font-medium text-[13px]"> </span>
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
