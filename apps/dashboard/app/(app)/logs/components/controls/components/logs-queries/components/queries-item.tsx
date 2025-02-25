// import { type PropsWithChildren, useEffect, useState } from "react";

import type { SavedFiltersGroup } from "@/app/(app)/logs/hooks/use-bookmarked-filters";
import { toast } from "@/components/ui/toaster";
import { cn } from "@/lib/utils";
import type { UserResource } from "@clerk/types";
import { Bookmark, ChartActivity2, Check, Clock, Conversion, Layers2, Link4 } from "@unkey/icons";
import { Tooltip, TooltipContent, TooltipTrigger } from "@unkey/ui";
import { Button } from "@unkey/ui";
import { format } from "date-fns";
import { type PropsWithChildren, useState } from "react";
import { QueriesMadeBy } from "./queries-made-by";
import { QueriesPill } from "./queries-pill";

type QueriesItemProps = {
  filterList: SavedFiltersGroup;
  user: UserResource | null | undefined;
  index: number;
  total: number;
  selectedIndex: number;
  querySelected: (index: number) => void;
  changeBookmark: (index: string) => void;
};
type QueriesToastProps = PropsWithChildren<{
  children: React.ReactNode;
  message: string;
}>;
export const QueriesToast = ({ children, message }: QueriesToastProps) => {
  return (
    <div className="flex flex-row items-center p-2 font-sans space-x-auto">
      {children}
      <span className="flex w-56 font-medium text-sm leading-6 text-accent-12 text-center justify-center items-center">
        {message}
      </span>
      <Button
        variant="ghost"
        className="flex end-0 p-2 m-0  rounded-[10px] border-[1px] border-gray-5 "
      >
        Undo
      </Button>
    </div>
  );
};
export const QueriesItem = ({
  filterList,
  user,
  index,
  total,
  selectedIndex,
  querySelected,
  changeBookmark,
}: QueriesItemProps) => {
  const { status, methods, paths, startTime, endTime } = filterList.filters;
  const startDateTime = startTime ? format(new Date(startTime * 1000), "MMM d HH:mm:ss.SS") : "";
  const endDateTime = endTime ? format(new Date(endTime * 1000), "MMM d HH:mm:ss.SS") : "";
  const timeValue = `${startDateTime} - ${endDateTime}`;
  const [isSaved, setIsSaved] = useState(filterList.bookmarked);
  const [tooltipMessage, setTooltipMessage] = useState<"Saved!" | "Save Query">(
    isSaved ? "Saved!" : "Save Query",
  );
  const [toolTipOpen, setToolTipOpen] = useState(false);
  const handleBookmarkChanged = () => {
    const newValue = !isSaved;
    setIsSaved(newValue);
    setTooltipMessage(newValue ? "Saved!" : "Save Query");
    changeBookmark(filterList.id);
    if (isSaved) {
      setTooltipMessage("Save Query");
      toast.success(
        <QueriesToast message="Query removed from Saved!">
          <Check className="size-[18px] text-success-9" />
        </QueriesToast>,
      );
    } else {
      setTooltipMessage("Saved!");
      toast.success(
        <QueriesToast message="Query Saved!">
          <Check className="size-[18px] text-success-9" />
        </QueriesToast>,
      );
    }
  };
  const handleSelection = (index: number) => {
    querySelected(index);
  };

  return (
    <div className="w-full">
      <div
        className={cn(
          "flex flex-row hover:bg-gray-2 cursor-pointer whitespace-nowrap rounded rounded-[8px] pb-[9px]",
          index === selectedIndex ? "bg-gray-2" : "",
        )}
      >
        <div
          className={cn("flex flex-col w-full", `tabIndex-${index}`)}
          role="button"
          onClick={() => handleSelection(index)}
          onKeyUp={(e) => e.key === "Enter" && console.log("clicked", index)}
          tabIndex={index}
        >
          {/* Change bg-gray-3 back to 2  */}
          <div className="w-full pt-[7px] px-[8px]">
            {/* Top Row for each */}
            <div className="flex flex-row justify-start items-center h-6">
              <div className="inline-flex gap-2 w-full">
                <span className="font-mono font-normal text-xs text-gray-9">from</span>
                <Layers2 className="size-3 mt-[1px]" />
                <span className="font-mono font-medium text-xs">Logs</span>
              </div>
            </div>

            {/* Filters */}
            <div className="flex flex-row mt-2">
              {/* Vertical Line on Left */}
              <div className="flex flex-col ml-[9px] border-l-[1px] border-l-gray-5 w-[1px]" />
              <div className="flex flex-col gap-2 ml-0 pl-[18px] ">
                {/* Map Thru each Status filter */}
                {status && status.length > 0 && (
                  <div className="flex flex-row justify-start items-center gap-2">
                    <div className="flex-col font-mono font-normal text-xs text-gray-9 align-start w-[43px]">
                      Status
                    </div>
                    <ChartActivity2 className="size-3.5 mb-[2px]" />
                    <span className="font-mono font-normal text-xs text-gray-9">
                      {status[0]?.operator}
                    </span>
                    {status?.map((item) => {
                      return <QueriesPill value={item.value} />;
                    })}
                  </div>
                )}
                {/* Map Thru each Method filter */}
                {methods && methods.length > 0 && (
                  <div className="flex flex-row justify-start items-center gap-2 ellipsis w-44">
                    <div className="flex-col font-mono font-normal text-xs text-gray-9 align-start w-[43px]">
                      Method
                    </div>
                    <Conversion className="size-3.5 mb-[2px] ml-[-1px]" />
                    <span className="font-mono font-normal text-xs text-gray-9">
                      {methods[0]?.operator}
                    </span>
                    {methods?.map((item) => {
                      return <QueriesPill value={item.value} />;
                    })}
                  </div>
                )}
                {/* Map Thru each Path filter */}
                {paths && paths.length > 0 && (
                  <div className="flex flex-row justify-start items-center gap-2 ">
                    <div className="flex-col font-mono font-normal text-xs text-gray-9 align-start w-[43px]">
                      Path
                    </div>
                    <Link4 className="size-3 ml-[1px]" />
                    <span className="font-mono font-normal text-xs text-gray-9">
                      {paths[0]?.operator}
                    </span>
                    <QueriesPill value={paths[0].value} />
                  </div>
                )}
                {startTime && (
                  <div className="flex flex-row justify-start items-center gap-2 w-56">
                    <div className="flex-col font-mono font-normal text-xs text-gray-9 align-start w-[43px]">
                      Time
                    </div>
                    <Clock className="size-3 ml-[1px]" />
                    <span className="font-mono font-normal text-xs text-gray-9">is</span>
                    <QueriesPill value={timeValue} />
                  </div>
                )}
                {/* {since && since.length > 0 && (
                  <div className="flex flex-row justify-start items-center gap-2">
                    <div className="flex-col font-mono font-normal text-xs text-gray-9 align-start w-[43px]">
                      Path
                    </div>
                    <Link4 className="size-3 ml-[1px]" />
                    <span className="font-mono font-normal text-xs text-gray-9">
                      {since[0]?.operator}
                    </span>
                    <QueriesPill value={paths[0].value} />
                  </div>
                )} */}
              </div>
            </div>
            <QueriesMadeBy
              userName={user?.username ?? ""}
              userImageSrc={user?.imageUrl ?? ""}
              createdString={"2 days ago"}
            />
          </div>
        </div>
        <div
          className="flex flex-col h-full pr-2 mt-1.5"
          onMouseEnter={() => setToolTipOpen(true)}
          onMouseLeave={() => setToolTipOpen(false)}
        >
          <Tooltip open={toolTipOpen}>
            <TooltipTrigger>
              <div
                className={cn(
                  "flex h-7 w-6 ml-[1px]  justify-center items-center text-accent-9 rounded-md",
                  isSaved ? "text-info-9 hover:bg-info-3" : "hover:bg-gray-3 hover:text-accent-12",
                  `tabIndex-${0}`,
                )}
                role="button"
                onClick={handleBookmarkChanged}
                onKeyUp={(e) => e.key === "Enter" && console.log("Saved", index)}
              >
                <Bookmark size="md-regular" filled={isSaved || false} />
              </div>
            </TooltipTrigger>
            <TooltipContent
              className="flex h-8 py-1 px-2 rounded-lg font-500 text-[12px] justify-center items-center leading-6 shadow-[0_12px_32px_-16px_rgba(0,0,0,0.3)] shadow-[0_12px_60px_1px_rgba(0,0,0,0.15)] shadow-[0_0px_0px_1px_rgba(0,0,0,0.1)]"
              side="bottom"
            >
              {tooltipMessage}
            </TooltipContent>
          </Tooltip>
        </div>
      </div>
      <div
        className={cn(
          "flex flex-row bg-white dark:bg-black h-[1px] mt-[7px] mb-[8px] w-full",
          index < total - 1 && "border-b-[1px] border-b-gray-3",
        )}
      />
    </div>
  );
};
