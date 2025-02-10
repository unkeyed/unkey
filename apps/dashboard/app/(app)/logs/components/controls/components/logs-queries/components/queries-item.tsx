// import { type PropsWithChildren, useEffect, useState } from "react";
import type { SavedFiltersGroup } from "@/app/(app)/logs/hooks/use-bookmarked-filters";
import { cn } from "@/lib/utils";
import { Bookmark, ChartActivity2, Conversion, Layers2, Link4 } from "@unkey/icons";
import { Tooltip, TooltipContent, TooltipTrigger } from "@unkey/ui";
import { useState } from "react";
import { QueriesMadeBy } from "./queries-made-by";
import { QueriesPill } from "./queries-pill";

type QueriesItemProps = {
  item: SavedFiltersGroup;
  index: number;
  total: number;
  selectedIndex: number;
  querySelected: (index: number) => void;
  changeBookmark: (index: number, isSaved: boolean) => void;
};

export const QueriesItem = ({
  item,
  index,
  total,
  selectedIndex,
  querySelected,
  changeBookmark,
}: QueriesItemProps) => {
  const { status, methods, paths } = item.filters;
  const [isSaved, setIsSaved] = useState(false);

  const handleBookmarkChanged = () => {
    setIsSaved((prev) => !prev);
    changeBookmark(index, isSaved);
  };

  return (
    <div
      className={cn("flex flex-col mt-2 w-[430px]")}
      role="button"
      onClick={() => querySelected(index)}
      onKeyUp={(e) => e.key === "Enter" && console.log("clicked", index)}
      tabIndex={index}
    >
      {/* Change bg-gray-3 back to 2  */}
      <div
        className={cn(
          "w-full p-2 hover:bg-gray-2 cursor-pointer whitespace-nowrap rounded rounded-4",
          index === selectedIndex ? "bg-gray-2" : "",
        )}
      >
        {/* Top Row for each */}
        <div className="flex flex-row justify-start items-center h-6">
          <div className="inline-flex gap-2 w-full">
            <span className="font-mono font-normal text-xs text-gray-9">from</span>
            <Layers2 className="size-3 " />
            <span className="font-mono font-medium text-xs">Logs</span>
          </div>
          <Tooltip>
            <TooltipTrigger>
              <div
                className="flex h-6 w-6 justify-center items-center hover:bg-gray-3 rounded-md"
                role="button"
                onClick={() => handleBookmarkChanged}
                onKeyUp={(e) => e.key === "Enter" && console.log("Saved", index)}
              >
                <Bookmark
                  filled={isSaved}
                  className={cn(
                    "size-3 text-accent-9 text-center hover:text-accent-12",
                    isSaved ? "text-info-9" : "",
                  )}
                />
              </div>
            </TooltipTrigger>
            <TooltipContent
              className="flex h-8 rounded-lg font-medium text-[12px] justify-center items-center leading-4"
              side="bottom"
            >
              Save Query
            </TooltipContent>
          </Tooltip>
        </div>
        {/* Filters */}
        <div className="flex flex-row mt-2">
          {/* Vertical Line on Left */}
          <div className="flex flex-col ml-[8px] border-l-[1px] border-l-gray-5 w-[1px]" />
          <div className="flex flex-col gap-2 ml-0 pl-[18px] ">
            {/* Map Thru each Status filter */}
            {status && status.length > 0 && (
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
      <div
        className={cn("bg-white h-2 w-full", index < total - 1 && "border-b-[1px] border-b-gray-3")}
      />
    </div>
  );
};
