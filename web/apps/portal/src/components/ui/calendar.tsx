import { ChevronLeft, ChevronRight } from "lucide-react";
import { DayPicker, type Matcher, useDayPicker } from "react-day-picker";
import { buttonVariants } from "~/components/ui/button";
import { cn } from "~/lib/utils";

const monthFormatter = new Intl.DateTimeFormat("en-US", {
  month: "long",
  year: "numeric",
});

function MonthCaption({ calendarMonth }: { calendarMonth: { date: Date } }) {
  const { goToMonth, nextMonth, previousMonth } = useDayPicker();
  return (
    <div className="mb-3.5 flex bg-transparent">
      <button
        type="button"
        disabled={!previousMonth}
        onClick={() => previousMonth && goToMonth(previousMonth)}
        className="flex min-h-6 min-w-6 items-center justify-center disabled:opacity-30"
      >
        <ChevronLeft className="size-3 text-gray-12" />
      </button>
      <div className="flex w-full items-center justify-center font-medium text-gray-12 text-sm">
        {monthFormatter.format(calendarMonth.date)}
      </div>
      <button
        type="button"
        disabled={!nextMonth}
        onClick={() => nextMonth && goToMonth(nextMonth)}
        className="flex min-h-6 min-w-6 items-center justify-center disabled:opacity-30"
      >
        <ChevronRight className="size-3 text-gray-12" />
      </button>
    </div>
  );
}

const dayPickerClassNames = {
  root: "flex w-full justify-center p-0 m-0",
  months: "flex flex-col w-full sm:flex-row sm:space-x-4 sm:space-y-0",
  month: "w-full p-0 mt-0",
  month_caption: "flex justify-between relative items-center",
  weeks: "flex flex-col w-full bg-transparent text-gray-12 gap-3.5 mt-3",
  weekdays: "flex w-full h-8 mx-0 px-0 pt-2 gap-3 justify-center items-center",
  weekday: "w-8 h-8 font-normal text-xs text-gray-8",
  week: "flex w-full justify-between",
  day: "h-8 w-8 text-center text-gray-12 rounded-sm text-sm p-0 relative focus:outline-hidden focus:ring-0 focus-within:relative focus-within:z-20",
  day_button: cn(
    buttonVariants({ variant: "ghost" }),
    "flex h-8 w-8 items-center justify-center rounded-sm p-0 font-normal text-[13px] text-gray-12 hover:bg-gray-3 focus:outline-hidden focus:ring-0 aria-selected:opacity-100",
  ),
  selected: "bg-gray-4 hover:bg-gray-3 hover:text-gray-10 focus:bg-gray-3 text-gray-12",
  today:
    "relative after:content-[''] after:absolute after:bottom-0.5 after:left-1/2 after:w-1 after:h-1 after:-translate-x-1/2 after:rounded-full after:bg-gray-10 text-gray-12 hover:bg-gray-3",
  outside:
    "text-gray-10 opacity-50 aria-selected:bg-gray-3 aria-selected:text-gray-12 aria-selected:opacity-100",
  disabled: "text-gray-10 hover:bg-transparent cursor-not-allowed",
  hidden: "invisible",
};

type CalendarProps = {
  className?: string;
  selected?: Date;
  onSelect?: (date: Date | undefined) => void;
  minDate?: Date;
  maxDate?: Date;
  showOutsideDays?: boolean;
};

export function Calendar({
  className,
  selected,
  onSelect,
  minDate,
  maxDate,
  showOutsideDays = true,
}: CalendarProps) {
  const disabled: Matcher[] = [];
  if (minDate) {
    disabled.push({ before: minDate });
  }
  if (maxDate) {
    disabled.push({ after: maxDate });
  }

  return (
    <DayPicker
      mode="single"
      selected={selected}
      onSelect={onSelect}
      hideNavigation
      showOutsideDays={showOutsideDays}
      disabled={disabled.length ? disabled : undefined}
      components={{ MonthCaption }}
      className={cn("", className)}
      classNames={dayPickerClassNames}
    />
  );
}
