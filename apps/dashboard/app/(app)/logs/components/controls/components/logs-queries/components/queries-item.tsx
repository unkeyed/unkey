// import { type PropsWithChildren, useEffect, useState } from "react";
import type { SavedFiltersGroup } from "@/app/(app)/logs/hooks/use-bookmarked-filters";
import { toast } from "@/components/ui/toaster";
import { cn } from "@/lib/utils";
import { Bookmark, ChartActivity2, Check, Conversion, Layers2, Link4 } from "@unkey/icons";
import { Tooltip, TooltipContent, TooltipTrigger } from "@unkey/ui";
import { useState } from "react";
import { QueriesMadeBy } from "./queries-made-by";
import { QueriesPill } from "./queries-pill";

type QueriesItemProps = {
  filterList: SavedFiltersGroup;
  index: number;
  total: number;
  selectedIndex: number;
  querySelected: (index: number) => void;
  changeBookmark: (index: number, isSaved: boolean) => void;
};
export const QueriesToast = () => {
  return (
    <div className="flex flex-row items-center gap-4 p-2 font-sans">
      <Check className="size-[18px] text-success-9" />
      <span className="w-full font-medium text-sm leading-6 text-accent-12">Query Saved!</span>
    </div>
  );
};
export const QueriesItem = ({
  filterList,
  index,
  total,
  selectedIndex,
  querySelected,
  changeBookmark,
}: QueriesItemProps) => {
  const { status, methods, paths } = filterList.filters;
  const [isSaved, setIsSaved] = useState(false);
  const [tooltipMessage, setTooltipMessage] = useState<"Saved!" | "Save Query">(
    isSaved ? "Saved!" : "Save Query",
  );
  const [toolTipOpen, setToolTipOpen] = useState(false);
  const handleBookmarkChanged = () => {
    if (!isSaved) {
      setIsSaved(!isSaved);
      setTooltipMessage("Saved!");
      toast.success(<QueriesToast />);
      changeBookmark(index, !isSaved);
    } else {
      setIsSaved(!isSaved);
      setTooltipMessage("Save Query");
      toast.success("Query no longer saved");
      changeBookmark(index, !isSaved);
    }

    changeBookmark(index, !isSaved);
  };
  const handleSelection = (index: number) => {
    querySelected(index);
  };

  return (
    <div className="mt-2 w-[430px]">
      <div
        className={cn(
          "flex flex-row hover:bg-gray-2 cursor-pointer whitespace-nowrap rounded rounded-[8px]",
          index === selectedIndex ? "bg-gray-2" : "",
        )}
      >
        <div
          className={cn("flex flex-col w-full")}
          role="button"
          onClick={() => handleSelection(index)}
          onKeyUp={(e) => e.key === "Enter" && console.log("clicked", index)}
          tabIndex={index}
        >
          {/* Change bg-gray-3 back to 2  */}
          <div className="w-full p-2">
            {/* Top Row for each */}
            <div className="flex flex-row justify-start items-center h-6">
              <div className="inline-flex gap-2 w-full">
                <span className="font-mono font-normal text-xs text-gray-9">from</span>
                <Layers2 className="size-3 " />
                <span className="font-mono font-medium text-xs">Logs</span>
              </div>
            </div>

            {/* Filters */}
            <div className="flex flex-row mt-2">
              {/* Vertical Line on Left */}
              <div className="flex flex-col ml-[8px] border-l-[1px] border-l-gray-5 w-[1px]" />
              <div className="flex flex-col gap-2 ml-0 pl-[18px] ">
                {/* Map Thru each Status filter */}
                {status && (
                  <div className="flex flex-row justify-start items-center gap-2">
                    <div className="flex-col font-mono font-normal text-xs text-gray-9 align-start w-11">
                      Status
                    </div>
                    <ChartActivity2 className="size-3" />
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
                  <div className="flex flex-row justify-start items-center gap-2">
                    <div className="flex-col font-mono font-normal text-xs text-gray-9 align-start w-11">
                      Method
                    </div>
                    <Conversion className="size-3" />
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
                  <div className="flex flex-row justify-start items-center gap-2">
                    <div className="flex-col font-mono font-normal text-xs text-gray-9 align-start w-11">
                      Path
                    </div>
                    <Link4 className="size-3" />
                    <span className="font-mono font-normal text-xs text-gray-9">
                      {paths[0]?.operator}
                    </span>
                    <QueriesPill value={paths[0].value} />
                  </div>
                )}
              </div>
            </div>
            <QueriesMadeBy
              userName={"chronark"}
              userImageSrc="/images/team/andreas.jpeg"
              createdString={"3d ago"}
            />
          </div>
        </div>
        <div
          className="flex flex-col h-6 w-6"
          onMouseEnter={() => setToolTipOpen(true)}
          onMouseLeave={() => setToolTipOpen(false)}
        >
          <Tooltip open={toolTipOpen}>
            <TooltipTrigger>
              <div
                className={cn(
                  "flex h-6 w-6 mr-2 mt-2 justify-center items-center text-accent-9 rounded-md",
                  isSaved ? "text-info-9 hover:bg-info-3" : "hover:bg-gray-3 hover:text-accent-12",
                )}
                role="button"
                onClick={handleBookmarkChanged}
                onKeyUp={(e) => e.key === "Enter" && console.log("Saved", index)}
              >
                <Bookmark filled={isSaved} className="size-[12px] text-center stroke-[1.5px]" />
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
          "flex flex-row bg-white h-2 w-full",
          index < total - 1 && "border-b-[1px] border-b-gray-3",
        )}
      />
    </div>
  );
};
