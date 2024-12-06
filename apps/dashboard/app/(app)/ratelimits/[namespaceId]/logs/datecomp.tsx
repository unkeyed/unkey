import { Calendar } from "@/components/ui/calendar";
import { Input } from "@/components/ui/input";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { Separator } from "@/components/ui/separator";
import { format, setHours, setMinutes } from "date-fns";
import React, { PropsWithChildren, useState } from "react";
import { SelectSingleEventHandler } from "react-day-picker";

type DateTimePickerProps = {
  date: Date;
  setDate: (date: Date) => void;
  calendarProps?: React.ComponentProps<typeof Calendar>;
  timeInputProps?: React.ComponentProps<typeof Input>;
  popoverContentProps?: React.ComponentProps<typeof PopoverContent>;
  timeInputLabel?: string;
  dateFormat?: string;
  className?: string;
  disabled?: boolean;
};

export function DateTimePicker({
  date,
  setDate,
  children,
  calendarProps,
  timeInputProps,
  popoverContentProps,
  timeInputLabel = "Time",
  dateFormat = "yyyy-MM-dd'T'HH:mm",
  className,
  disabled,
}: PropsWithChildren<DateTimePickerProps>) {
  const [selectedDateTime, setSelectedDateTime] = useState<Date>(date);

  const handleSelect: SelectSingleEventHandler = (day, selected) => {
    if (!selected) return;
    const hours = selectedDateTime.getHours();
    const minutes = selectedDateTime.getMinutes();
    const newDate = setMinutes(setHours(selected, hours), minutes);
    setSelectedDateTime(newDate);
    setDate(newDate);
  };

  const handleTimeChange: React.ChangeEventHandler<HTMLInputElement> = (e) => {
    const { value } = e.target;
    const [hours, minutes] = value.split(":").map(Number);
    const newDate = setMinutes(
      setHours(selectedDateTime, hours || 0),
      minutes || 0
    );
    setSelectedDateTime(newDate);
    setDate(newDate);
  };

  return (
    <Popover>
      <PopoverTrigger asChild disabled={disabled}>
        {children}
      </PopoverTrigger>
      <PopoverContent className="w-auto p-0" {...popoverContentProps}>
        <Calendar
          mode="single"
          selected={selectedDateTime}
          onSelect={handleSelect}
          initialFocus
          {...calendarProps}
        />
        <Separator />
        <div className="p-3">
          <label className="text-sm font-medium leading-none">
            {timeInputLabel}
          </label>
          <Input
            type="time"
            className="w-[260px] mt-2"
            onChange={handleTimeChange}
            disabled={disabled}
            {...timeInputProps}
          />
        </div>
      </PopoverContent>
    </Popover>
  );
}
