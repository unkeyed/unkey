import type { LogsFilterValue, QuerySearchParams } from "@/app/(app)/logs/filters.schema";
import { useBookmarkedFilters } from "@/app/(app)/logs/hooks/use-bookmarked-filters";
import type { SavedFiltersGroup } from "@/app/(app)/logs/hooks/use-bookmarked-filters";
import { useFilters } from "@/app/(app)/logs/hooks/use-filters";
import { KeyboardButton } from "@/components/keyboard-button";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { useKeyboardShortcut } from "@/hooks/use-keyboard-shortcut";
import { useUser } from "@clerk/nextjs";
import { type PropsWithChildren, useCallback, useState } from "react";
import { EmptyQueries } from "./empty";
import { ListGroup } from "./list-group";
import { QueriesTabs } from "./queries-tabs";

type QueriesPopoverProps = PropsWithChildren<{
  open: boolean;
  setOpen: (open: boolean) => void;
}>;
export const QueriesPopover = ({ open, setOpen, children }: QueriesPopoverProps) => {
  const [focusedTabIndex, setFocusedTabIndex] = useState(0);
  const [selectedQueryIndex, setSelectedQueryIndex] = useState(0);
  const { savedFilters, toggleBookmark } = useBookmarkedFilters();
  const { updateFilters } = useFilters();
  const { user } = useUser();
  const [filterGroups, setfilterGroups] = useState<SavedFiltersGroup[]>(savedFilters);
  const [isDisabled, setIsDisabled] = useState(savedFilters.length === 0);

  useCallback(() => {
    updateGroups(savedFilters);
  }, [savedFilters]);

  const updateGroups = (newGroups: SavedFiltersGroup[]) => {
    setfilterGroups(newGroups);
  };
  useKeyboardShortcut("q", () => {
    setIsDisabled(savedFilters.length === 0);
    if (!open && !isDisabled) {
      setOpen(true);
      setSelectedQueryIndex(0);
      setFocusedTabIndex(0);
    } else {
      setOpen(false);
      setFocusedTabIndex(0);
    }
  });

  const handleTabChange = (index: number) => {
    updateGroups(savedFilters);
    // changeTabGroupLists();
    setFocusedTabIndex(index);
  };

  const handleSelectedQuery = (index: number) => {
    const fieldList: QuerySearchParams = savedFilters[index].filters;
    const newFilters: LogsFilterValue[] = [];
    Object.entries(fieldList).forEach(([key, values]) => {
      if (typeof values !== "object") {
        return;
      }
      values?.forEach((value) =>
        newFilters.push({
          id: crypto.randomUUID(),
          field: key as keyof QuerySearchParams,
          operator: value.operator,
          value: value.value,
        }),
      );
    });

    if (newFilters) {
      updateFilters(newFilters);
    }
    setSelectedQueryIndex(index);
  };

  const handleBookmarkChanged = (groupId: string) => {
    const newGroups = toggleBookmark(groupId);
    updateGroups(newGroups);
  };

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild disabled={true}>
        {children}
      </PopoverTrigger>
      <PopoverContent
        onFocus={() => updateGroups(savedFilters)}
        className="flex flex-col w-[440px] h-[600px] max-h-[924px] bg-white dark:bg-black rounded-lg p-2 pb-0  shadow-shadow-black-a5 shadow-shadow-black-a3 shadow-shadow-inverted-2 dark:shadow-[0_12px_32px_-16px_rgba(255,255,255,0.1),0_12px_60px_0px_rgba(255,255,255,0.15),0_0px_0px_1px_rgba(255,255,255,0.1)] border-none shring-0"
        align="start"
        // onKeyDown={handleKeyNavigation}
      >
        <div className="flex flex-row w-full">
          <PopoverHeader />
        </div>
        <QueriesTabs selectedTab={focusedTabIndex} onChange={handleTabChange} />
        <div className="flex flex-col w-full h-full overflow-y-auto m-0 p-0 pt-[8px] scrollbar-hide">
          <EmptyQueries selectedTab={focusedTabIndex} list={filterGroups} />
          {focusedTabIndex === 0 &&
            filterGroups?.map((filterItem: SavedFiltersGroup, index: number) => {
              return (
                <ListGroup
                  key={filterItem.id}
                  user={user}
                  filterList={filterItem}
                  index={index}
                  total={filterGroups.length}
                  selectedIndex={selectedQueryIndex}
                  querySelected={handleSelectedQuery}
                  changeBookmark={handleBookmarkChanged}
                />
              );
            })}
          {focusedTabIndex === 1 &&
            filterGroups
              .filter((filter) => filter.bookmarked)
              .map((filterItem: SavedFiltersGroup, index: number) => {
                return (
                  <ListGroup
                    key={filterItem.id}
                    user={user}
                    filterList={filterItem}
                    index={index}
                    total={filterGroups.filter((filter) => filter.bookmarked).length}
                    selectedIndex={selectedQueryIndex}
                    querySelected={handleSelectedQuery}
                    changeBookmark={handleBookmarkChanged}
                  />
                );
              })}
        </div>
      </PopoverContent>
    </Popover>
  );
};

const PopoverHeader = () => {
  return (
    <div className="flex justify-between w-full h-8 ">
      <span className="text-text text-gray-9 text-[13px] w-full leading-6 text-normal tracking-[0.1px] mt-[4px] ml-[6px]">
        Select a query...
      </span>
      <KeyboardButton
        shortcut="Q"
        className="p-0 m-0 min-w-5 w-5 h-5 rounded-[5px] mt-1.5 mr-1.5"
      />
    </div>
  );
};
