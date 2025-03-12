import { KeyboardButton } from "@/components/keyboard-button";

import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { useKeyboardShortcut } from "@/hooks/use-keyboard-shortcut";
import { useUser } from "@clerk/nextjs";
import { type PropsWithChildren, useEffect, useRef, useState } from "react";
import { EmptyQueries } from "./empty";
import { ListGroup } from "./list-group";
import { QueriesTabs } from "./queries-tabs";


type SavedFiltersGroup<T> = {
  id: string;
  createdAt: number;
  filters: T;
  bookmarked?: boolean;
};

interface QueriesPopoverProps extends PropsWithChildren<{
  savedFilters: SavedFiltersGroup<any>[];
  toggleBookmark: (groupId: string) => void;
  applyFilterGroup: (id: string) => void;
  updateGroups: () => void;  
  
}> {}

export const QueriesPopover = ({ children, savedFilters, toggleBookmark, applyFilterGroup, updateGroups }: QueriesPopoverProps) => {
  // Get the user Data

  const { user } = useUser(); 
  const containerRef = useRef<HTMLDivElement>(null); 
  const [open, setOpen] = useState(false);
  const [focusedTabIndex, setFocusedTabIndex] = useState(0);
  const [selectedQueryIndex, setSelectedQueryIndex] = useState(0);
  // Things I need to handle here:
  // - Open/close the popover
  // - Handle tab change
  // - Handle selected query
  // - Handle bookmark changed  
  // - Track selected query index and focused tab index
  // - Handle keyboard navigation
  // - Handle toggling bookmark status with 'b' or 'B'
  

  

  const [filterGroups, setfilterGroups] = useState<SavedFiltersGroup<any>[]>(
    savedFilters.filter((filter) => filter),
  );

  const [savedGroups, setSavedGroups] = useState<SavedFiltersGroup<any>[]>(
    filterGroups.filter((filter) => filter.bookmarked),
  );

  const [isDisabled, setIsDisabled] = useState(savedFilters.length === 0);

  const handleUpdateGroups = () => {
    updateGroups();
  };

useEffect(() => {
  setfilterGroups(savedFilters);
  setSavedGroups(savedFilters.filter((filter) => filter.bookmarked));
  setIsDisabled(savedFilters.length === 0);
}, [savedFilters]);

  useKeyboardShortcut("q", () => {
    console.log({ savedFilters });
    
    setIsDisabled(savedFilters.length === 0);
    if (!open && !isDisabled) {
      setOpen(true);
      setSelectedQueryIndex(0);
      setFocusedTabIndex(0);
    } else {
      setOpen(false);
      setFocusedTabIndex(0);
      setSelectedQueryIndex(0);
    }
  });

  const handleTabChange = (index: number) => {
    setSelectedQueryIndex(0);
    setFocusedTabIndex(index);
    handleUpdateGroups();
  };

  const handleSelectedQuery = (id: string) => {
    const filterId = savedFilters.find((filter) => filter.id === id);
    const filterIndex = savedFilters.findIndex((filter) => filter.id === id);
    if (!filterId) {
      return;
    }
    applyFilterGroup(filterId.id);
    setSelectedQueryIndex(filterIndex);
  };

  const handleBookmarkChanged = (groupId: string) => {
    toggleBookmark(groupId);
    handleUpdateGroups();
  };

  const handleKeyNavigation = (e: React.KeyboardEvent) => {
    // Adjust scroll speed as needed

    if (containerRef.current) {
      const scrollSpeed = 50;
      // Handle up/down navigation
      if (e.key === "ArrowUp" || e.key === "k" || e.key === "K") {
        e.preventDefault();

        const currentList = focusedTabIndex === 0 ? filterGroups : savedGroups;
        const totalItems = currentList.length - 1;
        containerRef.current.scrollTop -= scrollSpeed;
        if (totalItems === 0) {
          return;
        }

        // Move selection up, wrap to bottom if at top
        setSelectedQueryIndex((prevIndex) => (prevIndex > 0 ? prevIndex - 1 : 0));
      } else if (e.key === "ArrowDown" || e.key === "j" || e.key === "J") {
        e.preventDefault();

        containerRef.current.scrollTop += scrollSpeed;
        const currentList = focusedTabIndex === 0 ? filterGroups : savedGroups;
        const totalItems = currentList.length - 1;

        if (totalItems === 0) {
          return;
        }

        // Move selection down, wrap to top if at bottom
        setSelectedQueryIndex((prevIndex) =>
          prevIndex < totalItems - 1 ? prevIndex + 1 : totalItems,
        );
      }
    }
    // Handle tab navigation
    if (e.key === "ArrowLeft" || e.key === "h" || e.key === "H") {
      // Move to All tab
      setFocusedTabIndex(0);
      setSelectedQueryIndex(0);
    } else if (e.key === "ArrowRight" || e.key === "l" || e.key === "L") {
      // Move to Saved tab
      setFocusedTabIndex(1);
      setSelectedQueryIndex(0);
    } else if (e.key === "Enter" || e.key === " ") {
      // Apply the selected filter
      const currentList = focusedTabIndex === 0 ? filterGroups : savedGroups;
      if (currentList.length > 0 && selectedQueryIndex < currentList.length) {
        handleSelectedQuery(currentList[selectedQueryIndex].id);
        setOpen(false);
      }
    }

    // Handle toggling bookmark status with 'b' or 'B'
    if (e.key === "b" || e.key === "B") {
      e.preventDefault();

      const currentList = focusedTabIndex === 0 ? filterGroups : savedGroups;
      if (currentList.length > 0 && selectedQueryIndex < currentList.length) {
        const selectedGroup = currentList[selectedQueryIndex];
        handleBookmarkChanged(selectedGroup.id);
      }
    }
  };

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild disabled={true}>
        {children}
      </PopoverTrigger>
      <PopoverContent
        onFocus={() => handleUpdateGroups()}
        className="flex flex-col min-w-[430px] w-full h-[924px] bg-white dark:bg-black rounded-lg p-2 pb-0 shadow-lg border-none"
        align="start"
        onKeyDown={handleKeyNavigation}
      >
        <div className="flex flex-row w-full">
          <PopoverHeader />
        </div>
        <QueriesTabs selectedTab={focusedTabIndex} onChange={handleTabChange} />
        <div
          className="flex flex-col w-full h-full overflow-y-scroll m-0 p-0 pt-[8px] scrollbar-hide"
          ref={containerRef}
        >
          <EmptyQueries
            selectedTab={focusedTabIndex}
            list={focusedTabIndex === 0 ? filterGroups : savedGroups}
          />
          {focusedTabIndex === 0 &&
            filterGroups?.map((filterItem: SavedFiltersGroup<any>, index: number) => {
              return (
                <ListGroup
                  key={filterItem.id}
                  user={user}
                  filterList={filterItem}
                  index={index}
                  total={filterGroups.length}
                  selectedIndex={selectedQueryIndex}
                  isSaved={filterItem.bookmarked}
                  querySelected={() => handleSelectedQuery(filterItem.id)}
                  changeBookmark={handleBookmarkChanged}
                />
              );
            })}
          {focusedTabIndex === 1 &&
            savedGroups.map((filterItem: SavedFiltersGroup<any>, index: number) => {
              return (
                <ListGroup
                  key={filterItem.id}
                  user={user}
                  filterList={filterItem}
                  index={index}
                  total={savedGroups.filter((filter) => filter.bookmarked).length}
                  selectedIndex={selectedQueryIndex}
                  isSaved={filterItem.bookmarked}
                  querySelected={() => handleSelectedQuery(filterItem.id)}
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
