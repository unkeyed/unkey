import { cn } from "@/lib/utils";
import { ChartBarAxisY } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { useState } from "react";
import { QueriesPopover } from "./components/queries-popover";

export const LogsQueries = () => {
  const [open, setOpen] = useState(false);
  return (
    <QueriesPopover open={open} setOpen={setOpen}>
      <div className="group">
        <Button
          variant="ghost"
          className={cn("flex flex-row group-data-[state=open]:bg-gray-4 p-0 m-0 h-[24px]")}
          aria-label="Log queries"
          aria-haspopup="true"
          title="Press 'Q' to toggle queries"
        >
          <div className="flex flex-row pt-[5px] pl-2 m-0 mr-0 pr-0 text-gray-9">
            <ChartBarAxisY />
          </div>
          <div className="p-0 m-0 pr-2 pl-0 mr-0">
            <span className="text-gray-12 font-medium text-[13px] leading-4">Queries</span>
          </div>
        </Button>
      </div>
    </QueriesPopover>
  );
};
