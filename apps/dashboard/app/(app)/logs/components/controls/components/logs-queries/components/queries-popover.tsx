import { useBookmarkedFilters } from "@/app/(app)/logs/hooks/use-bookmarked-filters";
import type { SavedFiltersGroup } from "@/app/(app)/logs/hooks/use-bookmarked-filters";
import { KeyboardButton } from "@/components/keyboard-button";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { useKeyboardShortcut } from "@/hooks/use-keyboard-shortcut";
import { type PropsWithChildren, useState } from "react";
import { QueriesItem } from "./queries-item";
import { QueriesTabs } from "./queries-tabs";

// import type { filterOutputSchema } from "@/app/(app)/logs/filters.schema";
// type filterType = {id: string, createdAt: number, daysAgoText: string, filters: typeof filterOutputSchema[], user: {name: string, url: string}};
const filters: SavedFiltersGroup[] = [
  {
    id: "1",
    createdAt: new Date().getTime(),
    filters: {
      status: [
        {
          value: "200",
          operator: "is",
        },
        {
          value: "400",
          operator: "is",
        },
        {
          value: "500",
          operator: "is",
        },
      ],
      methods: [
        {
          value: "GET",
          operator: "is",
        },
        {
          value: "PUT",
          operator: "is",
        },
        {
          value: "DELETE",
          operator: "is",
        },
      ],
      paths: [
        {
          value: "v1/keys.verifykey",
          operator: "is",
        },
      ],
      host: null,
      requestId: null,
    },
  },
  {
    id: "2",
    createdAt: new Date().getTime(),
    filters: {
      status: [
        {
          value: "400",
          operator: "is",
        },
        {
          value: "500",
          operator: "is",
        },
      ],
      methods: [
        {
          value: "GET",
          operator: "is",
        },
      ],
      paths: [],
      host: null,
      requestId: null,
    },
  },
  {
    id: "3",
    createdAt: new Date().getTime(),
    filters: {
      status: [
        {
          value: "500",
          operator: "is",
        },
      ],
      methods: [],
      paths: [],
      host: null,
      requestId: null,
    },
  },
  {
    id: "4",
    createdAt: new Date().getTime(),
    filters: {
      status: [],
      methods: [],
      paths: [
        {
          value: "v1/keys.verifykey",
          operator: "is",
        },
      ],
      host: null,
      requestId: null,
    },
  },
  {
    id: "5",
    createdAt: new Date().getTime(),
    filters: {
      status: [
        {
          value: "200",
          operator: "is",
        },
        {
          value: "400",
          operator: "is",
        },
        {
          value: "500",
          operator: "is",
        },
      ],
      methods: [
        {
          value: "GET",
          operator: "is",
        },
        {
          value: "PUT",
          operator: "is",
        },
        {
          value: "DELETE",
          operator: "is",
        },
      ],
      paths: [
        {
          value: "v1/keys.verifykey",
          operator: "is",
        },
      ],
      host: null,
      requestId: null,
    },
  },
  {
    id: "6",
    createdAt: new Date().getTime(),
    filters: {
      status: [
        {
          value: "500",
          operator: "is",
        },
      ],
      methods: [
        {
          value: "GET",
          operator: "is",
        },
        {
          value: "PUT",
          operator: "is",
        },
        {
          value: "DELETE",
          operator: "is",
        },
      ],
      paths: [
        {
          value: "v1/keys.verifykey",
          operator: "is",
        },
      ],
      host: null,
      requestId: null,
    },
  },
];

const users = [
  { name: "chronark", url: "/images/team/andreas.jpeg", since: "2d ago" },
  { name: "James Perkins", url: "/images/team/james.jpg", since: "3d ago" },
  { name: "chronark", url: "/images/team/andreas.jpeg", since: "4d ago" },
  { name: "James Perkins", url: "/images/team/james.jpg", since: "1w ago" },
  { name: "Oz", url: "/images/team/james.jpg", since: "1w ago" },
  { name: "chronark", url: "/images/team/andreas.jpeg", since: "2w ago" },
];

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
        className="flex flex-col w-[430px] bg-white dark:bg-black rounded-lg p-2 pb-0 h-[924px] shadow-[0_12px_32px_-16px_rgba(0,0,0,0.1)] shadow-[0_12px_60px_0px_rgba(0,0,0,0.15)] shadow-[0_0px_0px_1px_rgba(0,0,0,0.1)] border-none"
        align="start"
        // onKeyDown={handleKeyNavigation}
      >
        <div className="flex flex-row w-full">
          <PopoverHeader />
        </div>

        <QueriesTabs selectedTab={focusedTabIndex} onChange={setFocusedTabIndex} />
        <div className="flex flex-col w-full h-full overflow-y-auto m-0 p-0 pt-[8px] scrollbar-hide">
          {filters
            ? filters.map((filterItem, index) => (
                <QueriesItem
                  key={filterItem.id}
                  user={users[index]}
                  filterList={filterItem}
                  index={index}
                  total={filters.length}
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
    <div className="flex w-full h-8 justify-between ">
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
