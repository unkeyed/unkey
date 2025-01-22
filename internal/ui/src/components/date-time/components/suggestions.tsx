"use client";
// biome-ignore lint: React in this context is used throughout, so biome will change to types because no APIs are used even though React is needed.
import * as React from "react";
import { cn } from "../../../lib/utils";
import { useDateTimeContext } from "../date-time";

type SuggestionsProps = {
  className?: string;
  children?: React.ReactNode;
  suggestions: Array<{ label: string; relativeTime: number }>;
};
const DateTimeSuggestions: React.FC<SuggestionsProps> = ({ className, suggestions }) => {
  const { onStartTimeChange, onEndTimeChange, onDateChange } = useDateTimeContext();

  const handleClick = (timeFromNow: number) => {
    const endDate = new Date();
    const startDate = new Date(endDate.getTime() - timeFromNow);
    const range = { from: startDate, to: endDate };
    const startTime = {
      HH: startDate.getHours().toString(),
      mm: startDate.getMinutes().toString(),
      ss: startDate.getSeconds().toString(),
    };
    const endTime = {
      HH: endDate.getHours().toString(),
      mm: endDate.getMinutes().toString(),
      ss: endDate.getSeconds().toString(),
    };
    onDateChange(range);
    onStartTimeChange(startTime);
    onEndTimeChange(endTime);
  };
  return (
    <div className={cn("flex flex-col items-center justify-center gap-4 mt-2", className)}>
      {suggestions.map(({ label, relativeTime }) => (
        <button
          key={`${label}-${relativeTime}`}
          type="button"
          className="text-blue-10 text-xs font-medium"
          onClick={() => {
            handleClick(relativeTime);
          }}
        >
          {label}
        </button>
      ))}
    </div>
  );
};
export { DateTimeSuggestions };
