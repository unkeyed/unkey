import { ChevronLeft, ChevronRight } from "@unkey/icons";
import { format } from "date-fns";
import { useState } from "react";
// biome-ignore lint/correctness/noUnusedImports: otherwise biome complains
import React from "react";
import { type CaptionProps, type DateRange, DayPicker, useNavigation } from "react-day-picker";
import { cn } from "../../../lib/utils";
import { buttonVariants } from "../../button";
import { useDateTimeContext } from "../date-time";

function CustomCaptionComponent(props: CaptionProps) {
  const { goToMonth, nextMonth, previousMonth } = useNavigation();
  return (
    <div className="flex bg-transparent mb-3.5">
      <button
        className="flex min-w-6 min-h-6 justify-center items-center"
        type="button"
        disabled={!previousMonth}
        onClick={() => previousMonth && goToMonth(previousMonth)}
      >
        <ChevronLeft className="text-gray-12 size-3" />
      </button>
      <div className="flex w-full text-gray-12 justify-center items-center font-medium calendar-header">
        {format(props.displayMonth, "MMMM yyy")}
      </div>
      <button
        className="flex min-w-6 min-h-6 justify-center items-center"
        type="button"
        disabled={!nextMonth}
        onClick={() => nextMonth && goToMonth(nextMonth)}
      >
        <ChevronRight className="text-gray-12 size-3" />
      </button>
    </div>
  );
}

const styleClassNames = {
  root: "flex w-full justify-center p-0 m-0",
  tbody: "flex flex-col w-full bg-transparent w-full text-gray-12 gap-3.5 mt-3",
  months: "flex flex-col w-full sm:flex-row sm:space-x-4 sm:space-y-0",
  month: "w-full p-0 mt-0 ",
  caption: "flex justify-between relative items-center",
  caption_label: "text-sm font-medium",
  nav: "flex items-center",
  nav_button: cn(
    buttonVariants({ variant: "default" }),
    "h-7 w-7 opacity-50 hover:opacity-100 flex justify-center items-center",
  ),
  nav_button_previous: "absolute left-1",
  nav_button_next: "absolute right-1",
  table: "w-full bg-transparent border-none m-0 p-0 ",
  head_row:
    "flex flex-start w-full border-none h-8 mx-0 px-0 pt-2 gap-3 justify-center items-center",
  head_cell: "w-8 h-8 font-normal text-xs text-gray-8 bg-transparent border-none ",
  row: "flex w-full border-none justify-between ",
  cell: "border-none h-8 w-8 text-center text-gray-12 rounded rounded-3 text-sm p-0 relative focus:outline-none focus:ring-0 [&:has([aria-selected].day-outside)]:bg-gray-4 [&:has([aria-selected])]:bg-gray-4 focus-within:relative focus-within:z-20",
  day: cn(
    buttonVariants({ variant: "ghost" }),
    "h-8 w-8 p-0 font-normal aria-selected:opacity-100 text-[13px] flex items-center justify-center hover:bg-gray-3 text-gray-12 rounded rounded-3 focus:outline-none focus:ring-0",
  ),
  day_range_start: "hover:bg-gray-3 focus:bg-gray-5 text-gray-12",
  day_range_middle: "",
  day_range_end: "hover:bg-gray-2 focus:bg-gray-3 focus:text-gray-12",
  day_selected: "bg-gray-4 hover:bg-gray-3 hover:text-gray-10 focus:bg-gray-3 text-gray-12",
  day_today:
    "relative after:content-[''] after:absolute after:bottom-0.5 after:left-1/2 after:w-1 after:h-1 after:-translate-x-1/2 after:rounded-full after:bg-gray-10 text-gray-12 hover:bg-gray-3",
  day_outside:
    "day-outside text-gray-10 opacity-50 aria-selected:bg-gray-3 aria-selected:text-gray-12 aria-selected:opacity-100",
  day_disabled: "text-gray-10",
  day_hidden: "invisible",
};
type CalendarProps = {
  className?: string;
  classNames?: Record<string, string>;
  mode: "single" | "range";
  showOutsideDays?: boolean;
};

export const Calendar = ({
  className,
  classNames,
  mode,
  showOutsideDays = true,
  ...props
}: CalendarProps) => {
  const { date, onDateChange, minDateRange, maxDateRange } = useDateTimeContext();
  const [singleDay, setSingleDay] = useState<Date | undefined>(date?.from);

  const handleChange = (newDate: DateRange | undefined) => {
    //  No date selected (user cleared the selection)
    if (!newDate) {
      onDateChange({ from: undefined, to: undefined });
      return;
    }

    //  End date was moved later than current end date
    // This resets the "from" date while keeping the new "to" date
    if (
      date?.from &&
      date?.to &&
      newDate?.to instanceof Date &&
      newDate.to.getTime() > date.to.getTime()
    ) {
      onDateChange({ from: undefined, to: newDate.to });
      return;
    }

    // User clicked on either boundary of existing range
    // This resets the entire range when clicking on start or end date
    if (
      date?.from &&
      date?.to &&
      (newDate.from?.getTime() === date.from.getTime() ||
        newDate.from?.getTime() === date.to.getTime())
    ) {
      onDateChange({ from: undefined, to: undefined });
      return;
    }

    // Update with the new date range as is
    onDateChange(newDate);
  };

  const handleSingleChange = (newDate: Date | undefined) => {
    if (singleDay && newDate && singleDay.getTime() === newDate.getTime()) {
      setSingleDay(undefined);
      onDateChange({ from: undefined, to: undefined });
      return;
    }
    onDateChange({ from: newDate, to: undefined });
    setSingleDay(newDate);
  };

  const commonProps = {
    disabled: minDateRange && { before: minDateRange, after: maxDateRange },
    showOutsideDays,
    className: cn("", className),
    classNames: {
      ...styleClassNames,
      ...classNames,
    },
    components: {
      Caption: CustomCaptionComponent,
    },
    ...props,
  };

  if (mode === "range") {
    return (
      <DayPicker
        {...commonProps}
        mode="range"
        selected={date}
        disabled={(date) => date > new Date()}
        onSelect={(date: DateRange | undefined) => (date ? handleChange(date) : undefined)}
      />
    );
  }

  return (
    <DayPicker
      {...commonProps}
      mode="single"
      selected={singleDay}
      disabled={(date) => date > new Date()}
      onSelect={(date: Date | undefined) => handleSingleChange(date)}
    />
  );
};
