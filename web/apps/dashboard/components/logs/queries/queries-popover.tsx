import { useKeyboardShortcut } from "@/hooks/use-keyboard-shortcut";
import type { User } from "@/lib/auth/types";
import { useWorkspace } from "@/providers/workspace-provider";
import { IconBook2Outline18 } from "@unkey/icons";
import {
  Button,
  EmptyState,
  EmptyStateActions,
  EmptyStateDescription,
  EmptyStateHeader,
  EmptyStateTitle,
  KeyboardButton,
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@unkey/ui";
import { useEffect, useRef, useState } from "react";
import type { FilterValue } from "../validation/filter.types";
import { ListGroup } from "./list-group";
import { QueriesProvider, type QueryParamsTypes, useQueries } from "./queries-context";
import { QueriesTabs } from "./queries-tabs";

type QueriesPopoverProps<T extends FilterValue, U extends QueryParamsTypes> = {
  children: React.ReactNode;
  localStorageName: string;
  filters: T[];
  updateFilters: (filters: T[]) => void;
  formatFilterValues?: (
    filters: U,
  ) => Record<string, { operator: string; values: { value: string; color: string | null }[] }>;
  getFilterFieldIcon?: (field: string) => React.ReactNode;
  shouldTruncateRow?: (field: string) => boolean;
  fieldsToTruncate?: string[];
};

export function QueriesPopover<T extends FilterValue, U extends QueryParamsTypes>({
  children,
  localStorageName,
  filters,
  updateFilters,
  formatFilterValues,
  getFilterFieldIcon,
  shouldTruncateRow,
}: QueriesPopoverProps<T, U>) {
  const { user } = useWorkspace();
  const containerRef = useRef<HTMLDivElement>(null);
  const [open, setOpen] = useState(false);
  const [focusedTabIndex, setFocusedTabIndex] = useState(0);
  const [selectedQueryIndex, setSelectedQueryIndex] = useState(0);
  const [isDisabled, setIsDisabled] = useState(false);

  useKeyboardShortcut("q", () => {
    if (!open && !isDisabled) {
      setOpen(true);
      setSelectedQueryIndex(0);
      setFocusedTabIndex(0);
      setIsDisabled(filters.length === 0);
    } else {
      setIsDisabled(filters.length === 0);
      setOpen(false);
      setFocusedTabIndex(0);
      setSelectedQueryIndex(0);
    }
  });

  const handleTabChange = (index: number) => {
    setSelectedQueryIndex(0);
    setFocusedTabIndex(index);
  };

  const handleKeyNavigation = (e: React.KeyboardEvent) => {
    // Adjust scroll speed as needed
    if (containerRef.current) {
      const scrollSpeed = 50;
      // Handle up/down navigation
      if (e.key === "ArrowUp" || e.key === "k" || e.key === "K") {
        e.preventDefault();
        containerRef.current.scrollTop -= scrollSpeed;
      } else if (e.key === "ArrowDown" || e.key === "j" || e.key === "J") {
        e.preventDefault();
        containerRef.current.scrollTop += scrollSpeed;
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
      setOpen(false);
    }
  };

  return (
    <QueriesProvider<T, U>
      localStorageName={localStorageName}
      filters={filters}
      updateFilters={updateFilters}
      formatValues={formatFilterValues}
      filterRowIcon={getFilterFieldIcon}
      shouldTruncateRow={shouldTruncateRow}
    >
      <Popover open={open} onOpenChange={setOpen}>
        <PopoverTrigger render={children as React.ReactElement} />
        <PopoverContent
          className="flex flex-col min-w-107 max-w-200 h-[calc(100dvh-120px)] max-h-190 bg-white dark:bg-black rounded-lg p-2 pb-0 shadow-lg border-r border-gray-4"
          align="start"
          onKeyDown={handleKeyNavigation}
        >
          <div className="flex flex-row w-full">
            <PopoverHeader />
          </div>
          <QueriesTabs selectedTab={focusedTabIndex} onChange={handleTabChange} />
          <div
            className="flex flex-col w-full h-full overflow-y-scroll m-0 p-0 pt-2 scrollbar-hide"
            ref={containerRef}
          >
            <QueriesContent
              focusedTabIndex={focusedTabIndex}
              selectedQueryIndex={selectedQueryIndex}
              user={user}
            />
          </div>
        </PopoverContent>
      </Popover>
    </QueriesProvider>
  );
}

const PopoverHeader = () => {
  return (
    <div className="flex justify-between w-full h-8 ">
      <span className="text-text text-gray-9 text-[13px] w-full leading-6 text-normal tracking-[0.1px] mt-1 ml-1.5">
        Select a query...
      </span>
      <KeyboardButton
        shortcut="Q"
        className="p-0 m-0 min-w-5 w-5 h-5 rounded-[5px] mt-1.5 mr-1.5"
      />
    </div>
  );
};

type QueriesContentProps = {
  focusedTabIndex: number;
  selectedQueryIndex: number;
  user: User | undefined | null;
};

const QueriesContent = ({ focusedTabIndex, selectedQueryIndex, user }: QueriesContentProps) => {
  const { filterGroups, toggleBookmark, applyFilterGroup, formatValues } = useQueries();
  const [localFilterGroups, setLocalFilterGroups] = useState(filterGroups);

  useEffect(() => {
    setLocalFilterGroups(filterGroups);
  }, [filterGroups]);

  const handleSelectedQuery = (id: string) => {
    applyFilterGroup(id);
  };

  const handleBookmarkToggle = (id: string) => {
    toggleBookmark(id);
    setLocalFilterGroups((prev) =>
      prev.map((group) => (group.id === id ? { ...group, bookmarked: !group.bookmarked } : group)),
    );
  };

  const transformFilters = (filters: QueryParamsTypes) => {
    return formatValues(filters);
  };

  const isRecentTab = focusedTabIndex === 0;
  const isEmpty = isRecentTab
    ? localFilterGroups.length === 0
    : localFilterGroups.filter((filter) => filter.bookmarked).length === 0;

  return (
    <>
      {isEmpty && (
        <div className="flex items-center justify-between w-full h-full p-2 -mt-3.75">
          <EmptyState frame="none">
            <EmptyStateHeader>
              <EmptyStateTitle>
                {isRecentTab ? "No recent queries" : "No saved queries"}
              </EmptyStateTitle>
              <EmptyStateDescription>
                {isRecentTab
                  ? "Query using the filters, and they will show up here"
                  : "Save your recent queries and they will remain here"}
              </EmptyStateDescription>
            </EmptyStateHeader>
            <EmptyStateActions>
              <a
                href="https://www.unkey.com/docs/introduction"
                target="_blank"
                rel="noopener noreferrer"
              >
                <Button
                  variant="outline"
                  size="md"
                  className="flex items-center justify-center px-2"
                >
                  <IconBook2Outline18 className="py-0.5" />
                  Documentation
                </Button>
              </a>
            </EmptyStateActions>
          </EmptyState>
        </div>
      )}

      {focusedTabIndex === 0 &&
        localFilterGroups?.map((filterItem, index: number) => {
          return (
            <ListGroup
              key={filterItem.id}
              user={
                user
                  ? {
                      fullName: user.fullName ?? "",
                      imageUrl: user.avatarUrl ?? undefined,
                    }
                  : undefined
              }
              filterList={{
                filters: transformFilters(filterItem.filters),
                id: filterItem.id,
                createdAt: filterItem.createdAt,
                bookmarked: filterItem.bookmarked,
              }}
              index={index}
              total={localFilterGroups.length}
              selectedIndex={selectedQueryIndex}
              querySelected={() => handleSelectedQuery(filterItem.id)}
              changeBookmark={() => handleBookmarkToggle(filterItem.id)}
            />
          );
        })}

      {focusedTabIndex === 1 &&
        localFilterGroups
          .filter((filter) => filter.bookmarked)
          .map((filterItem) => {
            return (
              <ListGroup
                key={filterItem.id}
                user={
                  user
                    ? {
                        fullName: user.fullName ?? "",
                        imageUrl: user.avatarUrl ?? undefined,
                      }
                    : undefined
                }
                filterList={{
                  filters: transformFilters(filterItem.filters),
                  id: filterItem.id,
                  createdAt: filterItem.createdAt,
                  bookmarked: filterItem.bookmarked,
                }}
                index={localFilterGroups
                  .filter((f) => f.bookmarked)
                  .findIndex((f) => f.id === filterItem.id)}
                total={localFilterGroups.filter((f) => f.bookmarked).length}
                selectedIndex={selectedQueryIndex}
                querySelected={() => handleSelectedQuery(filterItem.id)}
                changeBookmark={() => handleBookmarkToggle(filterItem.id)}
              />
            );
          })}
    </>
  );
};
