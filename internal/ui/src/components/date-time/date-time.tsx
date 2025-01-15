"use client";

import { sub } from "date-fns";
import { createContext, useContext, useState } from "react";
// biome-ignore lint: React in this context is used throughout, so biome will change to types because no APIs are used even though React is needed.
import React from "react";
import type { DateRange } from "react-day-picker";
import { DateTimeActions } from "./components/actions";
import { Calendar } from "./components/calendar";
import { TimeSplitInput } from "./components/time-split";

export type DateTimeContextType = {
  minDateRange?: Date;
  maxDateRange?: Date;
  date?: DateRange;
  startTime?: TimeUnit;
  endTime?: TimeUnit;
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

type FullDateTime = {
  date: DateRange | undefined;
  startTime: TimeUnit | undefined;
  endTime: TimeUnit | undefined;
};
// Root Component
type DateTimeRootProps = {
  children: React.ReactNode;
  className?: string;
  value?: FullDateTime;
  minDate?: Date;
  maxDate?: Date;
  onChange: (
    value: DateRange | undefined,
    start: TimeUnit | undefined,
    end: TimeUnit | undefined,
  ) => void;
};

function DateTime({ children, className, value, minDate, maxDate, onChange }: DateTimeRootProps) {
  const today = new Date();
  const [minDateRange, setMinDate] = useState<Date>(minDate ?? sub(today, { years: 1 }));
  const [maxDateRange, setMaxDate] = useState<Date>(maxDate ?? today);
  const [date, setDate] = useState<DateRange>();
  const [startTime, setStartTime] = useState<TimeUnit>(
    value?.startTime || { HH: "00", mm: "00", ss: "00" },
  );
  const [endTime, setEndTime] = useState<TimeUnit>(
    value?.endTime || { HH: "00", mm: "00", ss: "00" },
  );

  const handleDateChange = (newRange: DateRange) => {
    setDate(newRange);
    onChange(newRange, startTime, endTime);
  };

  const handleStartTimeChange = (newTime: TimeUnit) => {
    setStartTime(newTime);
    onChange(date, newTime, endTime);
  };

  const handleEndTimeChange = (newTime: TimeUnit) => {
    setEndTime(newTime);
    onChange(date, startTime, newTime);
  };

  return (
    <div className={`flex flex-col gap-3 ${className}`}>
      <DateTimeContext.Provider
        value={{
          date,
          minDateRange,
          maxDateRange,
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

DateTime.TimeInput = TimeSplitInput;
DateTime.TimeInput.displayName = "DateTime.TimeInput";

DateTime.Actions = DateTimeActions;
DateTime.Actions.displayName = "DateTime.Actions";

export { DateTime, useDateTimeContext };
