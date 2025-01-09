import { useKeyboardShortcut } from "@/app/(app)/logs-v2/hooks/use-keyboard-shortcut";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { BarsFilter, CarretRight } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { type PropsWithChildren, useState } from "react";

type FilterItemConfig = {
  id: string;
  label: string;
  shortcut?: string;
};

const FILTER_ITEMS: FilterItemConfig[] = [
  {
    id: "status",
    label: "Status",
    shortcut: "s",
  },
  {
    id: "method",
    label: "Method",
    shortcut: "m",
  },
  {
    id: "path",
    label: "Path",
    shortcut: "p",
  },
];

const FilterItemContent = ({ id }: { id: string }) => {
  return (
    <div>
      <h3 className="text-sm font-medium mb-2">Filter by {id}</h3>
    </div>
  );
};

const FilterItem = ({ label, id, shortcut }: FilterItemConfig) => {
  const [open, setOpen] = useState(false);

  // Add keyboard shortcut for each filter item when main filter is open
  useKeyboardShortcut(
    { key: shortcut || "", meta: true },
    () => {
      setOpen(true);
    },
    { preventDefault: true }
  );

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <div className="flex w-full items-center px-2 py-1.5 justify-between hover:bg-gray-3 rounded-lg group cursor-pointer">
          <div className="flex gap-2 items-center">
            {shortcut && (
              <Button
                variant="ghost"
                tabIndex={-1}
                className="h-5 px-1.5 min-w-[24px] rounded bg-gray-3 text-gray-9 border-gray-8 border text-xs"
              >
                <div>
                  ⌘<span className="font-mono">{shortcut.toUpperCase()}</span>
                </div>
              </Button>
            )}

            <span className="text-[13px] text-accent-12 font-medium">
              {label}
            </span>
          </div>
          <div className="flex items-center gap-1.5">
            <Button
              variant="ghost"
              size="icon"
              tabIndex={-1}
              className="size-5 [&_svg]:size-2"
            >
              <CarretRight className="text-gray-7 group-hover:text-gray-10" />
            </Button>
          </div>
        </div>
      </PopoverTrigger>
      <PopoverContent
        className="min-w-60 bg-gray-1 dark:bg-black drop-shadow-2xl p-2 border-gray-6 rounded-lg"
        side="right"
        align="start"
        sideOffset={12}
      >
        <FilterItemContent id={id} />
      </PopoverContent>
    </Popover>
  );
};

const PopoverHeader = () => {
  return (
    <div className="flex w-full justify-between items-center px-2 py-1">
      <span className="text-gray-9 text-[13px]">Filters...</span>
      <Button
        variant="ghost"
        size="icon"
        tabIndex={-1}
        className="text-xs h-5 px-1.5 min-w-[24px] rounded bg-gray-3 text-gray-9 border-gray-8 border"
      >
        F
      </Button>
    </div>
  );
};

const FiltersPopover = ({ children }: PropsWithChildren) => {
  const [open, setOpen] = useState(false);

  useKeyboardShortcut("f", () => {
    setOpen((prev) => !prev);
  });

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>{children}</PopoverTrigger>
      <PopoverContent
        className="min-w-60 bg-gray-1 dark:bg-black drop-shadow-2xl p-2 border-gray-6 rounded-lg"
        align="start"
      >
        <div className="flex flex-col gap-2">
          <PopoverHeader />
          {FILTER_ITEMS.map((item) => (
            <FilterItem key={item.id} {...item} />
          ))}
        </div>
      </PopoverContent>
    </Popover>
  );
};

export const LogsFilters = () => {
  return (
    <FiltersPopover>
      <div className="group">
        <Button variant="ghost" className="group-data-[state=open]:bg-accent-4">
          <BarsFilter className="text-accent-9 size-4" />
          <span className="text-accent-12 font-medium text-[13px]">Filter</span>
        </Button>
      </div>
    </FiltersPopover>
  );
};
