"use client";
import { createContext, useContext, useState } from "react";
// biome-ignore lint: React in this context is used throughout
import React from "react";
import type { DateRange } from "react-day-picker";
import { DateTimeActions } from "./components/actions";
import { Calendar } from "./components/calendar";
import { TimeInput } from "./components/time-split";

export type DateTimeContextType = {
  minDate?: Date;
  maxDate?: Date;
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

type DateTimeRootProps = {
  children: React.ReactNode;
  className?: string;
  minDate?: Date;
  maxDate?: Date;
  initialRange?: DateRange;
  onChange: (date?: DateRange, start?: TimeUnit, end?: TimeUnit) => void;
};

function DateTime({
  children,
  className,
  onChange,
  initialRange,
  minDate,
  maxDate,
}: DateTimeRootProps) {
  const [date, setDate] = useState<DateRange | undefined>(initialRange);

  // Initialize time states based on initialRange dates if provided
  const [startTime, setStartTime] = useState<TimeUnit>(() => {
    if (initialRange?.from) {
      return {
        HH: initialRange.from.getHours().toString().padStart(2, "0"),
        mm: initialRange.from.getMinutes().toString().padStart(2, "0"),
        ss: initialRange.from.getSeconds().toString().padStart(2, "0"),
      };
    }
    return { HH: "00", mm: "00", ss: "00" };
  });

  const [endTime, setEndTime] = useState<TimeUnit>(() => {
    if (initialRange?.to) {
      return {
        HH: initialRange.to.getHours().toString().padStart(2, "0"),
        mm: initialRange.to.getMinutes().toString().padStart(2, "0"),
        ss: initialRange.to.getSeconds().toString().padStart(2, "0"),
      };
    }
    return { HH: "23", mm: "59", ss: "59" };
  });

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
    <DateTimeContext.Provider
      value={{
        date,
        startTime,
        endTime,
        minDate,
        maxDate,
        onDateChange: handleDateChange,
        onStartTimeChange: handleStartTimeChange,
        onEndTimeChange: handleEndTimeChange,
      }}
    >
      <div className={`flex flex-col w-80 justify-center items-center ${className}`}>
        {children}
      </div>
    </DateTimeContext.Provider>
  );
}

DateTime.displayName = "DateTime.root";
DateTime.Calendar = Calendar;
DateTime.TimeInput = TimeInput;
DateTime.TimeInput.displayName = "DateTime.TimeInput";
DateTime.Actions = DateTimeActions;
DateTime.Actions.displayName = "DateTime.Actions";

export { DateTime, useDateTimeContext };
