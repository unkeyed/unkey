import { useBookmarkedFilters } from "@/app/(app)/logs/hooks/use-bookmarked-filters";
import { useKeyboardShortcut } from "@/hooks/use-keyboard-shortcut";
import { KeyboardButton } from "@/components/keyboard-button";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { type PropsWithChildren, useState } from "react";
import { QueriesItem } from "./queries-item";
import { QueriesTabs } from "./queries-tabs";

export const QueriesPopover = ({ children }: PropsWithChildren) => {
  const [open, setOpen] = useState(false);
  const [focusedTabIndex, setFocusedTabIndex] = useState(1);
  const [selectedQueryIndex, setSelectedQueryIndex] = useState(0);
  const saved = useBookmarkedFilters();

  useKeyboardShortcut("Q", () => {
    setOpen((prev) => !prev);
    if (!open) {
      setOpen(true);
      setSelectedQueryIndex(0);
      setFocusedTabIndex(1);
      saved.savedFilters.map((item, index) => {
        console.log("Saved Filter", item, "Index", index);
      });
    } else {
      setOpen(false);
      setFocusedTabIndex(0);
    }
  });
  const handleBookmarkChanged = (index: number, isSaved: boolean) => {
    console.log("Bookmark changed index:", index, "Save State", isSaved);
  };

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <div className="flex flex-row items-center">{children}</div>
      </PopoverTrigger>
      <PopoverContent
        className="flex flex-col w-full bg-white dark:bg-black drop-shadow-3 radius-[8px] p-2 pb-0 max-h-[900px] shadow-[0_12px_32px_-16px_rgba(0,0,0,0.3)] shadow-[0_12px_60px_0px_rgba(0,0,0,0.15)] shadow-[0_0px_0px_1px_rgba(0,0,0,0.1)]"
        align="start"
        // onKeyDown={handleKeyNavigation}
      >
        <div className="flex flex-row w-full mt-[6px] px-[6px]">
          <PopoverHeader />
        </div>

        <QueriesTabs selectedTab={focusedTabIndex} onChange={setFocusedTabIndex} />
        <div className="flex flex-col w-full max-h-[700px] overflow-y-auto m-0 p-0 scrollbar-hide">
          {saved.savedFilters
            ? saved.savedFilters.map((filterItem, index) => (
                <QueriesItem
                  key={filterItem.id}
                  filterList={filterItem}
                  index={index}
                  total={saved.savedFilters.length}
                  selectedIndex={selectedQueryIndex}
                  querySelected={setSelectedQueryIndex}
                  changeBookmark={handleBookmarkChanged}
                />
              ))
            : null}
        </div>
      </PopoverContent>
    </Popover>
  );
};

const PopoverHeader = () => {
  return (
    <div className="flex w-full h-8 justify-between">
      <span className="text-gray-9 text-[13px] w-full leading-6">Select a query...</span>
      <KeyboardButton shortcut="Q" className="p-0 m-0 min-w-5 w-5 h-5" />
    </div>
  );
};
