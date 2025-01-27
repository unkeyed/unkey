// biome-ignore lint: React in this context is used throughout, so biome will change to types because no APIs are used even though React is needed.
import React, { useState } from "react";
import { buttonVariants } from "../../button";
import {
  type DateRange,
  DayPicker,
  IconLeft,
  IconRight,
  type CaptionProps,
  useNavigation,
} from "react-day-picker";
import { cn } from "../../../lib/utils";
import { format } from "date-fns";
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
        <IconLeft className="text-gray-12 size-2" />
      </button>
      <div className="flex w-full text-gray-12 text-sm justify-center items-center font-medium">
        {format(props.displayMonth, "MMMM yyy")}
      </div>
      <button
        className="flex min-w-6 min-h-6 justify-center items-center"
        type="button"
        disabled={!nextMonth}
        onClick={() => nextMonth && goToMonth(nextMonth)}
      >
        <IconRight className="text-gray-12 size-2" />
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
  row: "flex w-full border-none justify-between",
  cell: "border-none h-8 w-8 text-center rounded rounded-md text-sm p-0 relative [&:has([aria-selected].day-outside)]:bg-gray-4 [&:has([aria-selected])]:bg-gray-4  focus-within:relative focus-within:z-20",
  day: cn(
    buttonVariants({ variant: "ghost" }),
    "h-8 w-8 p-0 font-normal aria-selected:opacity-100 text-[13px] flex items-center justify-center hover:bg-gray-3",
  ),

  day_range_start: "hover:bg-gray-3 focus:bg-gray-5 text-gray-12",
  day_range_middle: "",
  day_range_end: "hover:bg-gray-2 focus:bg-gray-3 focus:text-gray-12",
  day_selected: "bg-gray-4 hover:bg-gray-3 hover:text-gray-10 focus:bg-gray-3 text-gray-12",
  day_today:
    "hover:aria-selected:bg-gray-3 hover:aria-selected:border-gray-8 bg-gray-12 text-gray-3 hover:text-gray-12 aria-selected:border aria-selected:border-2 aria-selected:border-gray-12 aria-selected:bg-gray-4 aria-selected:text-gray-12",
  day_outside:
    "day-outside text-gray-10 opacity-50 aria-selected:bg-gray-3 aria-selected:text-gray-12 aria-selected:opacity-100",
  day_disabled: "text-gray-10",
  day_hidden: "invisible",
};
type CalendarProps = {
  className?: string;
  classNames?: Record<string, string>;
  mode?: "single" | "range";
  showOutsideDays?: boolean;
};

const Calendar: React.FC<CalendarProps> = ({
  className,
  classNames,
  mode,
  showOutsideDays = true,
  ...props
}) => {
  const { date, onDateChange, minDateRange, maxDateRange } = useDateTimeContext();
  const [singleDay, setSingleDay] = useState<Date | undefined>(date?.from);
  const handleChange = (newDate: DateRange) => {
    onDateChange(newDate);
  };
  const handleSingleChange = (newDate: Date | undefined) => {
    onDateChange({ from: newDate, to: undefined });
    setSingleDay(newDate);
  };
  return mode === "range" ? (
    <DayPicker
      mode={mode}
      disabled={minDateRange && { before: minDateRange, after: maxDateRange }}
      selected={date}
      onSelect={(range: DateRange | undefined) => (range ? handleChange(range) : undefined)}
      showOutsideDays={showOutsideDays}
      className={cn("", className)}
      classNames={{
        ...styleClassNames,
        ...classNames,
      }}
      components={{
        Caption: CustomCaptionComponent,
      }}
      {...props}
    />
  ) : (
    <DayPicker
      mode="single"
      disabled={minDateRange && { before: minDateRange, after: maxDateRange }}
      selected={singleDay}
      onDayClick={(day: Date | undefined) => handleSingleChange(day)}
      showOutsideDays={showOutsideDays}
      className={cn("", className)}
      classNames={{
        ...styleClassNames,
        ...classNames,
      }}
      components={{
        Caption: CustomCaptionComponent,
      }}
      {...props}
    />
  );
};
export { Calendar };
