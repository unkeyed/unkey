import { toast } from "@/components/ui/toaster";
import { cn } from "@/lib/utils";
import type { UserResource } from "@clerk/types";

import { defaultFormatValue } from "@/components/logs/control-cloud/utils";
import { Bookmark, CircleCheck, Layers2 } from "@unkey/icons";
import { Tooltip, TooltipContent, TooltipTrigger } from "@unkey/ui";
import { useEffect, useState } from "react";
import { QueriesItemRow } from "./queries-item-row";
import { QueriesMadeBy } from "./queries-made-by";
import { QueriesToast } from "./queries-toast";
import { getSinceTime } from "./utils";
import { iconsPerField } from "./utils";
export type QuerySearchParams = {
  startTime?: number | null;
  endTime?: number | null;
  since?: string | null;
};

type SavedFiltersGroup<T> = {
  id: string;
  createdAt: number;
  filters: T;
  bookmarked?: boolean;
};

type ListGroupProps<T> = {
  filterList: SavedFiltersGroup<T>;
  user: UserResource | null | undefined;
  index: number;
  total: number;
  selectedIndex: number;
  isSaved: boolean | undefined;
  querySelected: (index: string) => void;
  changeBookmark: (index: string) => void;
};

type ToolTipMessageType =
  | "Saved!"
  | "Save Query"
  | "Remove query from Saved"
  | "Query removed from Saved!";

const tooltipMessageOptions: { [key: string]: ToolTipMessageType } = {
  saved: "Saved!",
  save: "Save Query",
  remove: "Remove query from Saved",
  removed: "Query removed from Saved!",
};

export function ListGroup<T extends QuerySearchParams>({
  filterList,
  user,
  index,
  total,
  selectedIndex,
  isSaved,
  querySelected,
  changeBookmark,
}: ListGroupProps<T>) {
  const [toolTipOpen, setToolTipOpen] = useState(false);
  const [saved, setSaved] = useState(isSaved);
  const [toastMessage, setToastMessage] = useState<ToolTipMessageType>();
  const [tooltipMessage, setTooltipMessage] = useState<ToolTipMessageType>(
    saved ? tooltipMessageOptions.saved : tooltipMessageOptions.save,
  );

  useEffect(() => {
    setSaved(isSaved);
  }, [isSaved]);
  let timeOperator = "since";
  const formatedFilters = () => {
    // Create a formatted version of each filter entry
    const formatted: Record<string, Array<{ operator: string; value: string }>> = {};
    // Initialize formatted with an empty time array
    formatted.time = [];

    // Determine timeOperator based on available time filters

    if (filterList.filters.startTime && filterList.filters.endTime) {
      timeOperator = "between";
    } else if (filterList.filters.startTime) {
      timeOperator = "starts from";
    }

    // Process each field in the filters
    Object.entries(filterList.filters).forEach(([field, value]) => {
      if (!value || (Array.isArray(value) && value.length === 0)) {
        return;
      }
      if (field === "startTime") {
        formatted.time.push({
          operator: timeOperator,
          value: defaultFormatValue(Number(value), field),
        });
      }
      if (field === "endTime") {
        formatted.time.push({
          operator: timeOperator,
          value: defaultFormatValue(Number(value), field),
        });
      }
      if (field === "since") {
        formatted.time.push({ operator: timeOperator, value: String(value) });
      }
      // Handle different types of values
      else if (Array.isArray(value)) {
        value.forEach((filter) => {
          if (!formatted[field]) {
            formatted[field] = [];
          }
          formatted[field].push({ operator: filter.operator, value: filter.value });
        });
      }
    });

    return formatted;
  };
  useEffect(() => {
    if (toastMessage) {
      handleToast(toastMessage);
    }
  }, [toastMessage]);

  const handleToast = (message: ToolTipMessageType) => {
    toast.success(
      <QueriesToast message={message} undoBookmarked={handleBookmarkChanged}>
        <CircleCheck size="xl-regular" className="text-success-9" />
      </QueriesToast>,
    );
  };

  const handleBookmarkChanged = () => {
    changeBookmark(filterList.id);
    const newIsSaved = !isSaved;
    const message = newIsSaved ? tooltipMessageOptions.saved : tooltipMessageOptions.removed;
    setToastMessage(message);
    setTooltipMessage(message);
  };

  const handleMouseEnter = () => {
    setToolTipOpen(true);
    setTooltipMessage(isSaved ? tooltipMessageOptions.remove : tooltipMessageOptions.save);
  };

  const handleMouseLeave = () => {
    setToolTipOpen(false);
  };

  const handleSelection = () => {
    querySelected(filterList.id);
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
          className={cn("flex flex-col w-11/12 ", `tabIndex-${index}`)}
          role="button"
          onClick={() => handleSelection()}
          onKeyUp={(e) => e.key === "Enter"}
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
                {Object.entries(formatedFilters()).map(([field, list]) => {
                  const Icon = iconsPerField[field] || Layers2;
                  // Choose icon based on field type
                  return (
                    <QueriesItemRow
                      key={field}
                      list={list}
                      field={field}
                      Icon={<Icon size="md-regular" className="justify-center" />}
                      operator={field === "time" ? timeOperator : "is"}
                    />
                  );
                })}
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
          onMouseEnter={handleMouseEnter}
          onMouseLeave={handleMouseLeave}
        >
          <Tooltip open={toolTipOpen}>
            <TooltipTrigger>
              <div
                className={cn(
                  "flex h-7 w-6 ml-[1px]  justify-center items-center text-accent-9 rounded-md",
                  saved ? "text-info-9 hover:bg-info-3" : "hover:bg-gray-3 hover:text-accent-12",
                  `tabIndex-${0}`,
                )}
                role="button"
                onClick={handleBookmarkChanged}
                onKeyUp={(e) => e.key === "Enter"}
                aria-label={saved ? "Remove from bookmarks" : "Add to bookmarks"}
              >
                <Bookmark size="md-regular" filled={saved || false} />
              </div>
            </TooltipTrigger>
            <TooltipContent
              className="flex h-8 py-1 px-2 rounded-lg font-500 text-[12px] justify-center items-center leading-6 shadow-lg"
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
}
