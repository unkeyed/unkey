import { ChevronLeft, ChevronRight } from "@unkey/icons";
import { format } from "date-fns";
// biome-ignore lint/correctness/noUnusedImports: Biome is not happy
import React, { useRef } from "react";
import {
  type CaptionProps,
  type DateRange,
  DayPicker,
  type Matcher,
  useNavigation,
} from "react-day-picker";
import { cn } from "../../../lib/utils";
import { buttonVariants } from "../../buttons/button";
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
    "h-8 w-8 p-0 font-normal aria-selected:opacity-100 text-[13px] flex items-center justify-center hover:bg-gray-3 text-gray-12 rounded rounded-3 text-sm focus:outline-none focus:ring-0",
  ),
  day_range_start: "hover:bg-gray-3 focus:bg-gray-5 text-gray-12",
  day_range_middle: "",
  day_range_end: "hover:bg-gray-2 focus:bg-gray-3 focus:text-gray-12",
  day_selected: "bg-gray-4 hover:bg-gray-3 hover:text-gray-10 focus:bg-gray-3 text-gray-12",
  day_today:
    "relative after:content-[''] after:absolute after:bottom-0.5 after:left-1/2 after:w-1 after:h-1 after:-translate-x-1/2 after:rounded-full after:bg-gray-10 text-gray-12 hover:bg-gray-3",
  day_outside:
    "day-outside text-gray-10 opacity-50 aria-selected:bg-gray-3 aria-selected:text-gray-12 aria-selected:opacity-100",
  day_disabled: "text-gray-10 hover:bg-transparent cursor-not-allowed",
  day_hidden: "invisible",
};

type CalendarProps = {
  className?: string;
  classNames?: Record<string, string>;
  mode: "single" | "range";
  showOutsideDays?: boolean;
  disabledDates?: Array<{
    from?: Date;
    to?: Date;
    before?: Date;
    after?: Date;
  }>;
};

export const Calendar = ({
  className,
  classNames,
  mode,
  showOutsideDays = true,
  disabledDates,
  ...props
}: CalendarProps) => {
  const { date, onDateChange, minDate, maxDate } = useDateTimeContext();

  const handleDayClick = (clickedDate: Date) => {
    const clickedTime = clickedDate.getTime();

    if (mode === "single") {
      // Toggle selection if same date clicked
      if (date?.from && date.from.getTime() === clickedTime) {
        onDateChange({ from: undefined, to: undefined });
        return;
      }
      onDateChange({ from: clickedDate, to: undefined });
      return;
    }

    // Range mode logic
    if (!date?.from) {
      // No selection, start new range
      onDateChange({ from: clickedDate, to: undefined });
      return;
    }

    if (!date.to) {
      // We have start date, complete the range
      const fromTime = date.from.getTime();

      if (clickedTime === fromTime) {
        // Clicked same start date, clear selection
        onDateChange({ from: undefined, to: undefined });
        return;
      }

      // Complete the range
      if (clickedTime < fromTime) {
        onDateChange({ from: clickedDate, to: date.from });
      } else {
        onDateChange({ from: date.from, to: clickedDate });
      }
      return;
    }

    // We have a complete range, start new selection
    onDateChange({ from: clickedDate, to: undefined });
  };

  // Only handle clears. User clicks are handled by handleDayClick
  // because react-day-picker's onSelect reconstructs ranges from the original start date
  // when clicking inside existing ranges, ignoring the actual clicked date.
  const handleRangeChange = (newRange: DateRange | undefined) => {
    if (!newRange) {
      onDateChange({ from: undefined, to: undefined });
    }
  };

  const handleSingleChange = (newDate: Date | undefined) => {
    if (!newDate) {
      onDateChange({ from: undefined, to: undefined });
    }
  };

  const getDisabledMatcher = (): Matcher | Matcher[] | undefined => {
    const matchers: Matcher[] = [];

    if (minDate) {
      matchers.push({ before: minDate });
    }

    if (maxDate) {
      matchers.push({ after: maxDate });
    }

    if (disabledDates?.length) {
      for (const dateRange of disabledDates) {
        if (dateRange.from && dateRange.to) {
          matchers.push({ from: dateRange.from, to: dateRange.to });
        } else if (dateRange.before) {
          matchers.push({ before: dateRange.before });
        } else if (dateRange.after) {
          matchers.push({ after: dateRange.after });
        }
      }
    }

    return matchers.length > 0 ? matchers : undefined;
  };

  const commonProps = {
    showOutsideDays,
    className: cn("", className),
    classNames: {
      ...styleClassNames,
      ...classNames,
    },
    components: {
      Caption: CustomCaptionComponent,
    },
    disabled: getDisabledMatcher(),
    onDayClick: handleDayClick,
    ...props,
  };

  if (mode === "range") {
    return <DayPicker {...commonProps} mode="range" selected={date} onSelect={handleRangeChange} />;
  }

  return (
    <DayPicker {...commonProps} mode="single" selected={date?.from} onSelect={handleSingleChange} />
  );
};
