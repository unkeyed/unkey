"use client";

import { createContext, useContext, useState } from "react";
// biome-ignore lint: React in this context is used throughout, so biome will change to types because no APIs are used even though React is needed.
import React from "react";
import type { DateRange } from "react-day-picker";
import { DateTimeActions } from "./components/actions";
import { Calendar } from "./components/calendar";
import { DateTimeSuggestions } from "./components/suggestions";
import { TimeInput } from "./components/time-split";
export type DateTimeContextType = {
  minDateRange?: Date;
  maxDateRange?: Date;
  date?: DateRange;
  startTime: TimeUnit;
  endTime: TimeUnit;
  onDateChange: (newDate: DateRange) => void;
  onStartTimeChange: (newTime: TimeUnit) => void;
  onEndTimeChange: (newTime: TimeUnit) => void;
};
export type Range = DateRange;
export type TimeUnit = {
  HH: string;
  mm: string;
  ss: string;
};

const DateTimeContext = createContext<DateTimeContextType>({} as DateTimeContextType);

const useDateTimeContext = () => {
  const context = useContext(DateTimeContext);
  if (!context) {
    throw new Error("DateTime components must be used within DateTime.Root");
  }
  return context;
};

// Root Component
type DateTimeRootProps = {
  children: React.ReactNode;
  className?: string;
  minDate?: Date;
  maxDate?: Date;
  onChange: (
    date: DateRange | undefined,
    start: TimeUnit | undefined,
    end: TimeUnit | undefined,
  ) => void;
};

function DateTime({ children, className, onChange }: DateTimeRootProps) {
  const [date, setDate] = useState<DateRange>();
  const [startTime, setStartTime] = useState<TimeUnit>({ HH: "00", mm: "00", ss: "00" });
  const [endTime, setEndTime] = useState<TimeUnit>({ HH: "00", mm: "00", ss: "00" });

  const handleDateChange = (newRange: DateRange) => {
    setDate(newRange);
    onChange(date, startTime, endTime);
  };

  const handleStartTimeChange = (newTime: TimeUnit) => {
    setStartTime(newTime);
    onChange(date, startTime, endTime);
  };

  const handleEndTimeChange = (newTime: TimeUnit) => {
    setEndTime(newTime);
    onChange(date, startTime, endTime);
  };

  return (
    <div className={`flex flex-row w-fit gap-3 ${className}`}>
      <DateTimeContext.Provider
        value={{
          date,
          startTime,
          endTime,
          onDateChange: handleDateChange,
          onStartTimeChange: handleStartTimeChange,
          onEndTimeChange: handleEndTimeChange,
        }}
      >
        {children}
      </DateTimeContext.Provider>
    </div>
  );
}

DateTime.displayName = "DateTime.root";

DateTime.Calendar = Calendar;
DateTime.Calendar.displayName = "DateTime.Calendar";

DateTime.TimeInput = TimeInput;
DateTime.TimeInput.displayName = "DateTime.TimeInput";

DateTime.Actions = DateTimeActions;
DateTime.Actions.displayName = "DateTime.Actions";

DateTime.Suggestions = DateTimeSuggestions;
DateTime.Suggestions.displayName = "DateTime.Suggestions";

export { DateTime, useDateTimeContext };
