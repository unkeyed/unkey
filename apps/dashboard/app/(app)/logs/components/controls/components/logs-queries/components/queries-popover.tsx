import { useBookmarkedFilters } from "@/app/(app)/logs/hooks/use-bookmarked-filters";
import { useKeyboardShortcut } from "@/app/(app)/logs/hooks/use-keyboard-shortcut";
import { KeyboardButton } from "@/components/keyboard-button";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { type PropsWithChildren, useState } from "react";
import { QueriesItem } from "./queries-item";
import { QueriesTabs } from "./queries-tabs";

export const QueriesPopover = ({ children }: PropsWithChildren) => {
  const [open, setOpen] = useState(false);
  const [focusedIndex, setFocusedIndex] = useState(0);
  const saved = useBookmarkedFilters();
  console.log("Saved Filters", saved);

  useKeyboardShortcut("d", () => {
    setOpen((prev) => !prev);
    if (!open) {
      setFocusedIndex(0);
    }
  });

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <div className="flex flex-row items-center">{children}</div>
      </PopoverTrigger>
      <PopoverContent
        className="flex flex-col w-full bg-gray-1 dark:bg-black drop-shadow-3 border-gray-6 rounded-lg p-2 pt-4"
        align="start"
        // onKeyDown={handleKeyNavigation}
      >
        <div className="flex flex-row w-full">
          <PopoverHeader />
        </div>

        <QueriesTabs selectedTab={focusedIndex} onChange={setFocusedIndex} />
        {saved.savedFilters
          ? saved.savedFilters.map((item, index) => (
              <QueriesItem
                key={item.id}
                item={item}
                index={index}
                total={saved.savedFilters.length}
              />
            ))
          : null}
      </PopoverContent>
    </Popover>
  );
};

const PopoverHeader = () => {
  return (
    <div className="flex w-full h-8 justify-between px-2">
      <span className="text-gray-9 text-[13px] w-full">Select a query...</span>
      <KeyboardButton shortcut="Q" className="p-0 m-0 min-w-5 w-5 h-5" />
    </div>
  );
};
