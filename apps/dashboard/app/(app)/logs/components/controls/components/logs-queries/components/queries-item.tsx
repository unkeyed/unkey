// import { type PropsWithChildren, useEffect, useState } from "react";
import type { SavedFiltersGroup } from "@/app/(app)/logs/hooks/use-bookmarked-filters";
import { cn } from "@/lib/utils";
import {
  Bookmark,
  ChartActivity2,
  CircleHalfDottedClock,
  Conversion,
  Layers2,
  Link4,
} from "@unkey/icons";
import { Tooltip, TooltipContent, TooltipTrigger } from "@unkey/ui";
import Image from "next/image";
import { useState } from "react";

type QueriesItemProps = {
  item: SavedFiltersGroup;
  index: number;
  total: number;
};

export const QueriesItem = ({ item, index, total }: QueriesItemProps) => {
  const { status, methods, paths } = item.filters;
  const [isSaved, setIsSaved] = useState(false);
  return (
    <div
      className={cn(
        "flex flex-col mt-2 pb-4",
        index < total - 1 && "border-b-[1px] border-b-gray-5 ",
      )}
      role="button"
      onClick={() => console.log("clicked", index)}
      onKeyUp={(e) => e.key === "Enter" && console.log("clicked", index)}
      tabIndex={index}
    >
      <div className="w-full px-2 hover:bg-gray-2 cursor-pointer whitespace-nowrap rounded rounded-4">
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
                onClick={() => setIsSaved(!isSaved)}
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
          <div className="flex flex-col ml-[8px] border-l-[1px] border-l-gray-5 w-[1px]"></div>
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
        <div className="flex flex-row justify-start items-center h-6 gap-2 mt-2">
          <span className="font-mono font-normal text-xs text-gray-9">by</span>
          <Image
            className="rounded-full border border-gray-4 border-[1px]"
            src="/images/team/andreas.jpeg"
            width={20}
            height={20}
            alt="Picture of the user"
          />
          <span className="font-mono font-medium leading-4 text-xs text-gray-12">chronark</span>
          <CircleHalfDottedClock className="size-3" />
          <span className="font-mono font-normal text-xs leading-4 text-gray-9">3d ago</span>
        </div>
      </div>
    </div>
  );
};
// type UserLineItemProps = {

// };
type StatusPillType = {
  value: string | number;
};
const QueriesPill = ({ value }: StatusPillType) => {
  let color = undefined;
  let wording = value;
  if (value === 200 || value === "200") {
    color = "bg-success-9";
    wording = "2xx";
  } else if (value === 400 || value === "400") {
    color = "bg-warning-9";
    wording = "4xx";
  } else if (value === 500 || value === "500") {
    color = "bg-error-9";
    wording = "5xx";
  }
  return (
    <div className="h-6 bg-gray-3 inline-flex justify-start items-center py-1.5 px-2 rounded rounded-md gap-2 ">
      {color && <div className={cn("w-2 h-2 rounded-[2px]", color)} />}
      <span className="font-mono font-medium text-xs text-gray-12 text-xs">{wording}</span>
    </div>
  );
};
