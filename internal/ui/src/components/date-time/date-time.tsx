"use client";

import { format, addDays, sub } from "date-fns";
import { createContext, useContext, useState } from "react";
// biome-ignore lint: React in this context is used throughout, so biome will change to types because no APIs are used even though React is needed.
import React from "react";
import { addToRange, isDateRange, isDayPickerDefault, isDayPickerRange, Matcher, type DateRange } from "react-day-picker";
import { Calendar } from "./components/calendar";
import { TimeInput } from "./components/time-split";

export type DateTimeContextType = {
  date?: DateRange;
  startTime?: string;
  endTime?: string;
  onDateChange: (newDate: DateRange) => void;
  onStartTimeChange: (newTime: string) => void;
  onEndTimeChange: (newTime: string) => void;
};
export type Range = DateRange;
type TimeUnit = {
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
  startTime: string | undefined;
  endTime: string | undefined;
};
// Root Component
type DateTimeRootProps = {
  children: React.ReactNode;
  className?: string;
  value?: FullDateTime;
  onChange: (value: DateRange) => void;
};

function DateTime({ children, className, value, onChange }: DateTimeRootProps) {
  // const [interimDate, setInterimDate] = useState<DateRange>({
  //     from: new Date(),
  //     to: new Date(),
  // });
  // const [finalDate, setFinalDate] = useState<DateRange>();
  const [date, setDate] = useState<DateRange>();
  const [startTime, setStartTime] = useState(value?.startTime || "09:00");
  const [endTime, setEndTime] = useState(value?.endTime || "17:00");

  
  const handleDateChange = (newRange: DateRange) => {
    setDate(newRange);
    onChange(newRange);
  };

  const handleStartTimeChange = (newTime: string) => {
    setStartTime(newTime);
  };

  const handleEndTimeChange = (newTime: string) => {
    setEndTime(newTime);
  };
  const isSameDay = (date1: Date, date2: Date) =>
    format(new Date(date1), "dd/MM/yyyy") === format(new Date(date2), "dd/MM/yyyy");

  const compareTimeUnits = (time1: TimeUnit, time2: TimeUnit): number => {
    const t1 = Number(time1.HH) * 3600 + Number(time1.mm) * 60 + Number(time1.ss);
    const t2 = Number(time2.HH) * 3600 + Number(time2.mm) * 60 + Number(time2.ss);
    return t1 - t2;
  };

  // const handleTimeConflicts = (normalizedTime: TimeUnit) => {
  //     // Only handle conflicts if start and end are on the same day
  //     if (!isSameDay(date?.from, date?.to)) return;

  //     // If this is a start time and it's later than the end time,
  //     // push the end time forward to match the start time
  //     if (type === "start" && compareTimeUnits(normalizedTime, endTime) > 0) {
  //         setEndTime(normalizedTime);
  //     }
  //     // If this is an end time and it's earlier than the start time,
  //     // pull the start time backward to match the end time
  //     else if (
  //         type === "end" &&
  //         compareTimeUnits(normalizedTime, startTime) < 0
  //     ) {
  //         setStartTime(normalizedTime);
  //     }
  // };
  return (
    <div className={`flex flex-col gap-3 ${className}`}>
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

export { DateTime, useDateTimeContext };
