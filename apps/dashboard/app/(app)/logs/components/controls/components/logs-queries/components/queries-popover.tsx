import { useBookmarkedFilters } from "@/app/(app)/logs/hooks/use-bookmarked-filters";
import { useKeyboardShortcut } from "@/app/(app)/logs/hooks/use-keyboard-shortcut";
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

  useKeyboardShortcut("d", () => {
    setOpen((prev) => !prev);
    if (!open) {
      setSelectedQueryIndex(0);
      setFocusedTabIndex(0);
    } else {
      setFocusedTabIndex(1);
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
        className="flex flex-col w-full bg-white dark:bg-black drop-shadow-3 border-gray-6 radius-4 p-2"
        align="start"
        // onKeyDown={handleKeyNavigation}
      >
        <div className="flex flex-row w-full mt-[6px] px-[6px]">
          <PopoverHeader />
        </div>

        <QueriesTabs selectedTab={focusedTabIndex} onChange={setFocusedTabIndex} />
        {saved.savedFilters
          ? saved.savedFilters.map((item, index) => (
              <QueriesItem
                key={item.id}
                item={item}
                index={index}
                total={saved.savedFilters.length}
                selectedIndex={selectedQueryIndex}
                querySelected={setSelectedQueryIndex}
                changeBookmark={handleBookmarkChanged}
              />
            ))
          : null}
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
