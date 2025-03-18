import { toast } from "@/components/ui/toaster";
import { cn } from "@/lib/utils";
import type { UserResource } from "@clerk/types";
import { Bookmark, CircleCheck, Layers2 } from "@unkey/icons";
import { Tooltip, TooltipContent, TooltipTrigger } from "@unkey/ui";
import { useEffect, useState } from "react";
import type { ParsedSavedFiltersType } from "../hooks/use-bookmarked-filters";
import { QueriesItemRow } from "./queries-item-row";
import { QueriesMadeBy } from "./queries-made-by";
import { QueriesToast } from "./queries-toast";
import { getSinceTime } from "./utils";

type ListGroupProps = {
  filterList: ParsedSavedFiltersType;
  user: UserResource | null | undefined;
  index: number;
  total: number;
  selectedIndex: number;
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

export function ListGroup({
  filterList,
  user,
  index,
  total,
  selectedIndex,
  querySelected,
  changeBookmark,
}: ListGroupProps) {
  const [toolTipOpen, setToolTipOpen] = useState(false);
  const [toastMessage, setToastMessage] = useState<ToolTipMessageType>();
  const [tooltipMessage, setTooltipMessage] = useState<ToolTipMessageType>(
    filterList.bookmarked ? tooltipMessageOptions.saved : tooltipMessageOptions.save,
  );

  useEffect(() => {
    if (toastMessage) {
      handleToast(toastMessage);
    }
  }, [toastMessage]);

  const handleToast = (message: ToolTipMessageType) => {
    toast.success(
      <QueriesToast message={message} undoBookmarked={() => changeBookmark(filterList.id)}>
        <CircleCheck size="xl-regular" className="text-success-9" />
      </QueriesToast>,
    );
  };

  function handleBookmarkChanged() {
    const newIsSaved = !filterList.bookmarked;
    const message = newIsSaved ? tooltipMessageOptions.saved : tooltipMessageOptions.removed;
    setToastMessage(message);
    setTooltipMessage(message);
    changeBookmark(filterList.id);
  }

  const handleMouseEnter = () => {
    setToolTipOpen(true);
    setTooltipMessage(
      filterList.bookmarked ? tooltipMessageOptions.remove : tooltipMessageOptions.save,
    );
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
                {filterList &&
                  Object.entries(filterList.filters).map(([field, filter]) => {
                    const { values, operator, icon } = filter;
                    const Icon = icon || Layers2;
                    return (
                      <QueriesItemRow
                        key={field}
                        list={values}
                        field={field}
                        Icon={<Icon size="md-regular" className="justify-center" />}
                        operator={operator}
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
                  filterList.bookmarked
                    ? "text-info-9 hover:bg-info-3"
                    : "hover:bg-gray-3 hover:text-accent-12",
                  `tabIndex-${0}`,
                )}
                role="button"
                onClick={() => handleBookmarkChanged()}
                onKeyUp={(e) => e.key === "Enter"}
                aria-label={filterList.bookmarked ? "Remove from bookmarks" : "Add to bookmarks"}
              >
                <Bookmark size="md-regular" filled={filterList.bookmarked} />
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
