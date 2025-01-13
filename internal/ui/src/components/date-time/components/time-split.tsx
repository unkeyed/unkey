import { format } from "date-fns";
import { Clock } from "lucide-react";
import { useState, useContext } from "react";
import { DateTimeContext } from "../date-time";
// biome-ignore lint: React in this context is used throughout, so biome will change to types because no APIs are used even though React is needed.
import * as React from "react";

type TimeUnit = {
    HH: string;
    mm: string;
    ss: string;
};

type TimeInputType = "start" | "end";
type TimeField = keyof TimeUnit;

export interface TimeInputProps {
    type: TimeInputType;
    className?: string;
    
} 

const MAX_VALUES = {
    HH: 23,
    mm: 59,
    ss: 59,
} as const;

const TimeInput: React.FC<TimeInputProps> = ({
    type,
    className,
}) => {

    const {startTime, endTime,date, onStartTimeChange, onEndTimeChange} = useContext(DateTimeContext);
    const [focus, setFocus] = useState(false);
    
    const normalizeTimeUnit = (time: TimeUnit): TimeUnit => ({
        HH: time.HH.padStart(2, "0") || "00",
        mm: time.mm.padStart(2, "0") || "00",
        ss: time.ss.padStart(2, "0") || "00",
    });

    const isSameDay = (date1: Date, date2: Date) =>
        format(new Date(date1), "dd/MM/yyyy") ===
        format(new Date(date2), "dd/MM/yyyy");

    const compareTimeUnits = (time1: TimeUnit, time2: TimeUnit): number => {
        const t1 =
            Number(time1.HH) * 3600 + Number(time1.mm) * 60 + Number(time1.ss);
        const t2 =
            Number(time2.HH) * 3600 + Number(time2.mm) * 60 + Number(time2.ss);
        return t1 - t2;
    };

    const handleTimeConflicts = (normalizedTime: TimeUnit) => {
        if(date?.to && date?.from){
            const startDate = date.from;
            const endDate = date.to;
            if (!isSameDay(startDate, endDate)) return;

            // If this is a start time and it's later than the end time,
            // push the end time forward to match the start time
            if (type === "start" && compareTimeUnits(normalizedTime, endTime) > 0) {
                setEndTime(normalizedTime);
            }
            // If this is an end time and it's earlier than the start time,
            // pull the start time backward to match the end time
            else if (
                type === "end" &&
                compareTimeUnits(normalizedTime, startTime) < 0
            ) {
                setStartTime(normalizedTime);
            }
        }
        // Only handle conflicts if start and end are on the same day
       
    };

    const handleBlur = () => {
        const normalizedTime = normalizeTimeUnit(time);
        setTime(normalizedTime);
        handleTimeConflicts(normalizedTime);
        setFocus(false);
    };

    const handleChange = (value: string, field: TimeField) => {
        if (value.length > 2) return;

        const numValue = Number(value);
        if (value && numValue > MAX_VALUES[field]) return;

        setTime({ ...time, [field]: value });
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
            className={`
        flex h-7 w-fit items-center justify-center px-4
        gap-0 rounded border border-strong
        text-xs text-foreground-light
        ${focus ? "border-stronger outline outline-2 outline-border" : ""}
      `}
        >
            <div className="mr-1 text-foreground-lighter">
                <Clock size={14} strokeWidth={1.5} />
            </div>

            {renderTimeInput("HH", "Hours")}
            <span className="text-foreground-lighter">:</span>
            {renderTimeInput("mm", "Minutes")}
            <span className="text-foreground-lighter">:</span>
            {renderTimeInput("ss", "Seconds")}
        </div>
    );
};

export { TimeInput };
