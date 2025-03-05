// import { type PropsWithChildren, useEffect, useState } from "react";
import type { SavedFiltersGroup } from "@/app/(app)/logs/hooks/use-bookmarked-filters";
import { toast } from "@/components/ui/toaster";
import { cn } from "@/lib/utils";
import type { UserResource } from "@clerk/types";
import { Bookmark, Check, Layers2 } from "@unkey/icons";
import { Tooltip, TooltipContent, TooltipTrigger } from "@unkey/ui";
import {
  differenceInDays,
  differenceInHours,
  differenceInMinutes,
  differenceInMonths,
  differenceInSeconds,
  differenceInWeeks,
  differenceInYears,
} from "date-fns";
import { useState } from "react";
import { QueriesMadeBy } from "./queries-made-by";
import { MethodRow } from "./queries-method-row";
import { PathRow } from "./queries-path-row";
import { StatusRow } from "./queries-status-row";
import { TimeRow } from "./queries-time-row";
import { QueriesToast } from "./queries-toast";

type ListGroupProps = {
  filterList: SavedFiltersGroup;
  user: UserResource | null | undefined;
  index: number;
  total: number;
  selectedIndex: number;
  querySelected: (index: number) => void;
  changeBookmark: (index: string) => void;
};
type ToolTipMessageType =
  | "Saved!"
  | "Save Query"
  | "Remove query from Saved"
  | "Query removed from Saved!";
const tooltopMessageOptions: { [key: string]: ToolTipMessageType } = {
  saved: "Saved!",
  save: "Save Query",
  remove: "Remove query from Saved",
  removed: "Query removed from Saved!",
};

export const ListGroup = ({
  filterList,
  user,
  index,
  total,
  selectedIndex,
  querySelected,
  changeBookmark,
}: ListGroupProps) => {
  const { status, methods, paths, startTime, endTime, since } = filterList.filters;

  const [isSaved, setIsSaved] = useState(filterList.bookmarked);
  const [tooltipMessage, setTooltipMessage] = useState<ToolTipMessageType>(
    isSaved ? tooltopMessageOptions.saved : tooltopMessageOptions.save,
  );

  const [toolTipOpen, setToolTipOpen] = useState(false);
  const handleBookmarkChanged = () => {
    const newValue = !isSaved;
    setIsSaved(newValue);
    setTooltipMessage(newValue ? tooltopMessageOptions.saved : tooltopMessageOptions.removed);
    changeBookmark(filterList.id);
    if (isSaved) {
      setTooltipMessage(tooltopMessageOptions.removed);
      toast.success(
        <QueriesToast message={tooltopMessageOptions.removed}>
          <Check className="size-[18px] text-success-9" />
        </QueriesToast>,
      );
    } else {
      setTooltipMessage(tooltopMessageOptions.saved);
      toast.success(
        <QueriesToast message={tooltopMessageOptions.saved}>
          <Check className="size-[18px] text-success-9" />
        </QueriesToast>,
      );
    }
  };

  const handleMouseEnter = () => {
    setToolTipOpen(true);
    isSaved
      ? setTooltipMessage(tooltopMessageOptions.remove)
      : setTooltipMessage(tooltopMessageOptions.save);
  };
  const handleMouseLeave = () => {
    setToolTipOpen(false);
    // isSaved ? setTooltipMessage("Saved!") : setTooltipMessage("Save Query");
  };
  const handleSelection = (index: number) => {
    querySelected(index);
  };

  return (
    <div className="w-full">
      <div
        className={cn(
          "flex flex-row hover:bg-gray-2 cursor-pointer whitespace-nowrap rounded-[8px] pb-[9px] w-full",
          index === selectedIndex ? "bg-gray-2" : "",
        )}
      >
        <div
          className={cn("flex flex-col w-11/12", `tabIndex-${index}`)}
          role="button"
          onClick={() => handleSelection(index)}
          onKeyUp={(e) => e.key === "Enter" && console.log("clicked", index)}
          tabIndex={index}
        >
          <div className=" pt-[7px] px-[8px]">
            {/* Top Row for each */}
            <div className="flex flex-row items-center justify-start h-6">
              <div className="inline-flex w-full gap-2">
                <span className="font-mono text-xs font-normal text-gray-9">from</span>
                <Layers2 className="size-3 mt-[1px]" />
                <span className="font-mono text-xs font-medium">Logs</span>
              </div>
            </div>

            {/* Filters */}
            <div className="flex flex-row w-full mt-2">
              {/* Vertical Line on Left */}
              <div className="flex flex-col ml-[9px] border-l-[1px] border-l-gray-5 w-[1px]" />
              <div className="flex flex-col gap-2 ml-0 pl-[18px] ">
                {/* Status filter row*/}
                <StatusRow status={status} />
                {/* Method filter row*/}
                <MethodRow methods={methods} />
                {/* Path filter row*/}
                <PathRow paths={paths} />
                {/* Time filter row*/}
                <TimeRow startTime={startTime} endTime={endTime} since={since} />
              </div>
            </div>
            <QueriesMadeBy
              userName={user?.username ?? ""}
              userImageSrc={user?.imageUrl ?? ""}
              createdString={getSinceTime(filterList.createdAt)}
            />
          </div>
        </div>
        <div
          className="flex flex-col h-[24px] pr-2 mt-1.5 w-[24px]"
          onMouseEnter={() => handleMouseEnter()}
          onMouseLeave={() => handleMouseLeave()}
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
              className="flex h-8 py-1 px-2 rounded-lg font-500 text-[12px] justify-center items-center leading-6 shadow-[0_12px_32px_-16px_rgba(0,0,0,0.3),0_12px_60px_1px_rgba(0,0,0,0.15)],0_0px_0px_1px_rgba(0,0,0,0.1)]"
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

const getSinceTime = (date: number) => {
  const now = new Date();
  const seconds = differenceInSeconds(now, date);
  if (seconds < 60) {
    return "just now";
  }
  const minutes = differenceInMinutes(now, date);
  if (minutes < 60) {
    return `${minutes}m ago`;
  }
  const hours = differenceInHours(now, date);
  if (hours < 24) {
    return `${hours}h ago`;
  }
  const days = differenceInDays(now, date);
  if (days < 7) {
    return `${days}d ago`;
  }

  const weeks = differenceInWeeks(now, date);
  if (weeks < 4) {
    return `${weeks}w ago`;
  }

  const months = differenceInMonths(now, date);
  if (months < 12) {
    return `${months} month(s) ago`;
  }

  const years = differenceInYears(now, date);
  return `${years} year(s) ago`;
};
