// biome-ignore lint: React in this context is used throughout, so biome will change to types because no APIs are used even though React is needed.
import React, { useState } from "react";
import { buttonVariants } from "../../button";
import { DateRange, DayPicker, DayPickerRangeProps, IconLeft, IconRight, CaptionProps, useNavigation } from "react-day-picker";
import { cn } from "../../../lib/utils";
import { format } from "date-fns";
import { useDateTimeContext } from "../date-time";
// import "react-day-picker/style.css";
function CustomCaptionComponent(props: CaptionProps) {
    const { goToMonth, nextMonth, previousMonth } = useNavigation();
    return (
        <div className="flex flex-row w-full bg-transparent px-2">
            <button disabled={!previousMonth} onClick={() => previousMonth && goToMonth(previousMonth)}>
                <IconLeft className="text-gray-12 size-2" />
            </button>
            <div className="w-full text-center text-gray-12 text-xs">
                {format(props.displayMonth, "MMMM yyy")}
            </div>
            <button disabled={!nextMonth} onClick={() => nextMonth && goToMonth(nextMonth)}>
                <IconRight className="text-gray-12 size-2" />
            </button>
        </div>
    );
}

const Calendar: React.FC<DayPickerRangeProps> = ({ className, classNames, mode, showOutsideDays = true, ...props }) => {
    const { date, onDateChange } = useDateTimeContext();
    const handleChange = (newDate: DateRange) => {
        onDateChange(newDate);
    }
    return (
        <DayPicker
            mode="range"
            selected={date}
            onSelect={(range: DateRange | undefined) => range ? handleChange(range) : undefined}
            showOutsideDays={showOutsideDays}
            className={cn("", className)}
            classNames={{
                root: "flex justify-center p-2",
                tbody: "flex flex-col bg-transparent gap-y-2 w-full text-gray-12 mt-2",
                months: "flex flex-col sm:flex-row sm:space-x-4 sm:space-y-0 mt-0 pt-0",
                month: "w-full p-0 mt-0",
                caption: "flex justify-between px-2 relative items-center",
                caption_label: "text-sm font-medium",
                nav: "flex items-center my-0 py-0",
                nav_button: cn(
                    buttonVariants({ variant: "default" }),
                    "h-7 w-7 opacity-50 hover:opacity-100 flex justify-center items-center",
                ),
                nav_button_previous: "absolute left-1",
                nav_button_next: "absolute right-1",
                table: "w-full bg-transparent border-none mt-2",
                head_row: "flex border-none h-8 mt-0 pt-0",
                head_cell: "w-8 h-8 font-normal text-xs text-gray-8 bg-transparent border-none",
                row: "flex w-full border-none ",
                cell: "border-none h-8 w-8 text-center rounded text-sm p-0 relative [&:has([aria-selected].day-range-end)]:rounded-r-md [&:has([aria-selected].day-outside)]:bg-accent/50 [&:has([aria-selected])]:bg-accent first:[&:has([aria-selected])]:rounded-l-md last:[&:has([aria-selected])]:rounded-r-md focus-within:relative focus-within:z-20",
                day: cn(
                    buttonVariants({ variant: "ghost" }),
                    "h-8 w-8 p-0 font-normal aria-selected:opacity-100 text-[13px] flex items-center justify-center",
                ),

                day_range_start: "bg-gray-4 hover:bg-primary hover:text-primary-foreground focus:bg-primary focus:text-primary-foreground",
                day_range_middle:
                    "bg-blue-800",
                day_range_end: "bg-gray-4 hover:bg-primary hover:text-primary-foreground focus:bg-primary focus:text-primary-foreground",
                day_selected:
                    "bg-gray-4 hover:bg-primary hover:text-primary-foreground focus:bg-primary focus:text-primary-foreground",
                day_today: "bg-gray-12 text-gray-1",
                day_outside:
                    "day-outside text-gray-10 opacity-50 aria-selected:bg-accent/50 aria-selected:text-gray-12 aria-selected:opacity-100",
                day_disabled: "text-muted-foreground opacity-50",

                day_hidden: "invisible",
                ...classNames,
            }}
            components={{
                Caption: CustomCaptionComponent,
            }}
            {...props}
           
        />
    );
}
export { Calendar };