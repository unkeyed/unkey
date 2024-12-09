import { Calendar } from "@/components/ui/calendar";
import { Input } from "@/components/ui/input";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { Separator } from "@/components/ui/separator";
import { format, setHours, setMinutes } from "date-fns";
import type React from "react";
import { type PropsWithChildren, useState } from "react";

type DateTimePickerProps = {
  date: Date;
  onDateChange: (date: Date) => void;
  calendarProps?: React.ComponentProps<typeof Calendar>;
  timeInputProps?: React.ComponentProps<typeof Input>;
  popoverContentProps?: React.ComponentProps<typeof PopoverContent>;
  timeInputLabel?: string;
  disabled?: boolean;
};

export function DateTimePicker({
  date,
  onDateChange: setDate,
  children,
  calendarProps,
  timeInputProps,
  popoverContentProps,
  timeInputLabel = "Time",
  disabled,
}: PropsWithChildren<DateTimePickerProps>) {
  const [selectedDateTime, setSelectedDateTime] = useState<Date>(date);

  const handleSelect = (selected: Date) => {
    if (!selected) {
      return;
    }
    const hours = selectedDateTime.getHours();
    const minutes = selectedDateTime.getMinutes();
    const newDate = setMinutes(setHours(selected, hours), minutes);
    setSelectedDateTime(newDate);
    setDate(newDate);
  };

  const handleTimeChange: React.ChangeEventHandler<HTMLInputElement> = (e) => {
    const { value } = e.target;
    const [hours, minutes] = value.split(":").map(Number);
    const newDate = setMinutes(setHours(selectedDateTime, hours || 0), minutes || 0);
    setSelectedDateTime(newDate);
    setDate(newDate);
  };

  const timeValue = format(selectedDateTime, "HH:mm");

  return (
    <Popover>
      <PopoverTrigger asChild disabled={disabled}>
        {children}
      </PopoverTrigger>
      <PopoverContent className="w-auto p-0" {...popoverContentProps}>
        <Calendar
          mode="single"
          selected={selectedDateTime}
          //@ts-expect-error Calendar can't infer the mode correctly
          onSelect={(date) => handleSelect(date ?? new Date())}
          initialFocus
          {...calendarProps}
        />
        <Separator />
        <div className="p-3 flex flex-col gap-2">
          <label className="text-sm font-medium leading-none ml-0.5">{timeInputLabel}</label>
          <Input
            type="time"
            value={timeValue}
            className="w-[130px] mt-2"
            onChange={handleTimeChange}
            disabled={disabled}
            {...timeInputProps}
          />
        </div>
      </PopoverContent>
    </Popover>
  );
}
