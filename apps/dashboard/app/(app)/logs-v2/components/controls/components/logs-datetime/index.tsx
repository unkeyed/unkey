import { cn } from "@/lib/utils";
import { Calendar } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { useState } from "react";
import { DatetimePopover } from "./components/datetime-popover";

export const LogsDateTime = () => {
  const [title, setTitle] = useState<string>("Last 12 hours");
  const [isSelected, setIsSelected] = useState<boolean>(false);

  return (
    <DatetimePopover setTitle={setTitle} setSelected={setIsSelected}>
      <div className="group">
        <Button
          variant="ghost"
          className={cn("group-data-[state=open]:bg-gray-4 px-2", isSelected ? "bg-gray-4" : "")}
          aria-label="Filter logs by time"
          aria-haspopup="true"
          title="Press 'T' to toggle filters"
        >
          <Calendar className="text-gray-9 size-4" />
          <span className="text-gray-12 font-medium text-[13px]">{title}</span>
        </Button>
      </div>
    </DatetimePopover>
  );
};
