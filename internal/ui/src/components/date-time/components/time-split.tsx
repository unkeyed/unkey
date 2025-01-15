import { format } from "date-fns";

import { TimeClock } from "@unkey/icons";
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
    className?: string;
    inputClassNames?: string;
    type: TimeInputType;
};
const MAX_VALUES = {
    HH: 23,
    mm: 59,
    ss: 59,
} as const;

const TimeSplitInput: React.FC<TimeSplitInputProps> = ({ type, className }) => {
    const { startTime, endTime, date, onStartTimeChange, onEndTimeChange } = useDateTimeContext();
    const [focus, setFocus] = useState(false);
    const [time, setTime] = useState<TimeUnit>({ HH: "00", mm: "00", ss: "00" });

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
        if (date && date.from && date.to) {
            if (!isSameDay(date.from, date?.to)) return;
        }
        // If this is a start time and it's later than the end time,
        // push the end time forward to match the start time
        if (type === "start" && endTime && compareTimeUnits(normalizedTime, endTime) > 0) {
            onStartTimeChange(normalizedTime);
        }
        // If this is an end time and it's earlier than the start time,
        // pull the start time backward to match the end time
        else if (type === "end" && startTime && compareTimeUnits(normalizedTime, startTime) < 0) {
            onEndTimeChange(normalizedTime);
        }
    };

    const handleBlur = () => {
        const normalizedTime = normalizeTimeUnit(time);
        setTime(normalizedTime);
        handleTimeConflicts(normalizedTime);
        if (type === "start") {
            onStartTimeChange(normalizedTime);
            console.log("New Start Time:", normalizedTime);
        }
        if (type === "end") {
            onEndTimeChange(normalizedTime);
            console.log("New End Time:", normalizedTime);
        }
        setFocus(false);
    };

    const handleChange = (value: string, field: TimeField) => {
        if (value.length > 2) return;

        const numValue = Number(value);
        if (value && numValue > MAX_VALUES[field]) return;
        setTime({ ...time, [field]: value });
        console.log("New Start Time:", time);
    };

    const handleFocus = (event: React.FocusEvent<HTMLInputElement>) => {
        event.target.select();
        setFocus(true);
    };

    const inputClassNames = `
    w-4 p-0 
    border-none bg-transparent
    text-xs text-center text-foreground
    outline-none ring-0 focus:ring-0
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
                className,
                `
        flex h-7 w-fit items-center justify-center px-4
        gap-0 rounded border border-strong
        text-xs text-foreground-light
        ${focus ? "border-stronger outline outline-2 outline-border" : ""}
      `,
            )}
        >
            <div className="text-gray-9 mr-2">
                <TimeClock className="size-2.5" />
            </div>
            {renderTimeInput("HH", "Hours")}
            <span className="text-foreground-lighter">:</span>
            {renderTimeInput("mm", "Minutes")}
            <span className="text-foreground-lighter">:</span>
            {renderTimeInput("ss", "Seconds")}
            <span className="text-foreground-lighter"> </span>
            {/* AM/PM and timezone still needs to be implemented */}
            {/* {renderTimeInput("")} */}
        </div>
    );
};

export { TimeSplitInput };
