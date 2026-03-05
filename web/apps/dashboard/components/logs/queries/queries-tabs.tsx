"use client";
import { cn } from "@/lib/utils";
import { Bookmark, ClockRotateClockwise } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { useEffect, useState } from "react";

type QueriesTabsProps = {
  selectedTab: number;
  onChange: (index: number) => void;
};

export const QueriesTabs = ({ selectedTab, onChange }: QueriesTabsProps) => {
  const [selected, setSelected] = useState(selectedTab);

  useEffect(() => {
    handleSelection(selectedTab);
  }, [selectedTab]);

  const handleSelection = (index: number) => {
    setSelected(index);
    onChange(index);
  };

  return (
    <div className="flex mt-2 h-[40px] flex-row justify-center items-center w-full border-b border-gray-6 p-0 m-0 gap-2 shrink-0">
      <Button
        variant="ghost"
        className={cn(
          "h-full bg-base-12 rounded-b-none w-full ml-0 pl-[10px] focus:bg-accent-3 focus:ring-0 cursor-pointer",
          selected === 0 ? "bg-accent-3" : "",
        )}
        type="button"
        aria-label="Log queries"
        aria-haspopup="false"
        title="Recent queries"
        onClick={() => handleSelection(0)}
        onFocus={() => handleSelection(0)}
        onBlur={() => handleSelection(0)}
      >
        <ClockRotateClockwise iconSize="md-medium" className="text-gray-9 py-px" />
        <div className="w-full">Recent</div>
        <div
          className={cn(
            "absolute bottom-0 w-full bg-transparent h-[2px] pb-0 mb-0 ml-[2px]",
            selected === 0 ? "bg-accent-12" : "",
          )}
        />
      </Button>
      <Button
        variant="ghost"
        className={cn(
          "h-full bg-base-12 rounded-b-none w-full cursor-pointer focus:bg-accent-3 focus:ring-0",
          selectedTab === 1 ? "bg-accent-3" : "",
        )}
        type="button"
        aria-label="Log queries"
        aria-haspopup="false"
        title="Saved queries"
        onClick={() => handleSelection(1)}
        onFocus={() => handleSelection(1)}
        onBlur={() => handleSelection(1)}
      >
        <div className="w-4 h-4 text-gray-9">
          <Bookmark iconSize="sm-regular" className="text-gray-9 py-[1.5px]" />
        </div>
        <div className="w-full">Saved</div>
        <div
          className={cn(
            "absolute bottom-0 w-full bg-transparent h-[2px] pb-0 mb-0 ",
            selected === 1 ? "bg-accent-12" : "",
          )}
        />
      </Button>
    </div>
  );
};
