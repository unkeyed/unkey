import { useFilters } from "@/app/(app)/logs-v2/hooks/use-filters";
import { useKeyboardShortcut } from "@/app/(app)/logs-v2/hooks/use-keyboard-shortcut";
import { KeyboardButton } from "@/components/keyboard-button";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { CaretRight } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { type KeyboardEvent, type PropsWithChildren, useEffect, useRef, useState } from "react";
import { MethodsFilter } from "./methods-filter";
import { PathsFilter } from "./paths-filter";
import { StatusFilter } from "./status-filter";

type FilterItemConfig = {
  id: string;
  label: string;
  shortcut?: string;
  component: React.ReactNode;
};

const FILTER_ITEMS: FilterItemConfig[] = [
  {
    id: "status",
    label: "Status",
    shortcut: "s",
    component: <StatusFilter />,
  },
  {
    id: "methods",
    label: "Method",
    shortcut: "m",
    component: <MethodsFilter />,
  },
  {
    id: "paths",
    label: "Path",
    shortcut: "p",
    component: <PathsFilter />,
  },
];

export const FiltersPopover = ({ children }: PropsWithChildren) => {
  const [open, setOpen] = useState(false);
  const [focusedIndex, setFocusedIndex] = useState<number | null>(null);
  const [activeFilter, setActiveFilter] = useState<string | null>(null);

  useKeyboardShortcut("f", () => {
    setOpen((prev) => !prev);
  });

  const handleKeyDown = (e: KeyboardEvent) => {
    if (!open) {
      return;
    }

    // If we have an active filter and press left, close it
    if ((e.key === "ArrowLeft" || e.key === "h") && activeFilter) {
      e.preventDefault();
      setActiveFilter(null);
      return;
    }

    switch (e.key) {
      case "ArrowDown":
      case "j":
        e.preventDefault();
        setFocusedIndex((prev) => (prev === null ? 0 : (prev + 1) % FILTER_ITEMS.length));
        break;
      case "ArrowUp":
      case "k":
        e.preventDefault();
        setFocusedIndex((prev) =>
          prev === null
            ? FILTER_ITEMS.length - 1
            : (prev - 1 + FILTER_ITEMS.length) % FILTER_ITEMS.length,
        );
        break;
      case "Enter":
      case "l":
      case "ArrowRight":
        e.preventDefault();
        if (focusedIndex !== null) {
          const selectedFilter = FILTER_ITEMS[focusedIndex];
          if (selectedFilter) {
            // Find the filterItem component and trigger its open state
            const filterRefs = document.querySelectorAll("[data-filter-id]");
            const selectedRef = filterRefs[focusedIndex] as HTMLElement;
            if (selectedRef) {
              selectedRef.click();
              setActiveFilter(selectedFilter.id);
            }
          }
        }
        break;
      case "h":
      case "ArrowLeft":
        // Don't handle left arrow in main popover - let it bubble to FilterItem
        break;
    }
  };

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>{children}</PopoverTrigger>
      <PopoverContent
        className="w-60 bg-gray-1 dark:bg-black drop-shadow-2xl p-2 border-gray-6 rounded-lg"
        align="start"
        onKeyDown={handleKeyDown}
      >
        <div className="flex flex-col gap-2">
          <PopoverHeader />
          {FILTER_ITEMS.map((item, index) => (
            <FilterItem key={item.id} {...item} isFocused={focusedIndex === index} />
          ))}
        </div>
      </PopoverContent>
    </Popover>
  );
};

const PopoverHeader = () => {
  return (
    <div className="flex w-full justify-between items-center px-2 py-1">
      <span className="text-gray-9 text-[13px]">Filters...</span>
      <KeyboardButton shortcut="F" />
    </div>
  );
};

type FilterItemProps = FilterItemConfig & {
  isFocused?: boolean;
};

export const FilterItem = ({ label, shortcut, id, component, isFocused }: FilterItemProps) => {
  const { filters } = useFilters();
  const [open, setOpen] = useState(false);
  const itemRef = useRef<HTMLDivElement>(null);

  const handleKeyDown = (e: KeyboardEvent) => {
    if ((e.key === "ArrowLeft" || e.key === "h") && open) {
      e.preventDefault();
      setOpen(false);
      itemRef.current?.focus();
    }
  };

  useKeyboardShortcut(
    { key: shortcut || "", meta: true },
    () => {
      setOpen(true);
    },
    { preventDefault: true },
  );

  // Focus the element when isFocused changes
  useEffect(() => {
    if (isFocused && itemRef.current) {
      itemRef.current.focus();
    }
  }, [isFocused]);

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <div
          ref={itemRef}
          className={`flex w-full items-center px-2 py-1.5 justify-between rounded-lg group cursor-pointer
            hover:bg-gray-3 data-[state=open]:bg-gray-3 focus:outline-none
            ${isFocused ? "bg-gray-3" : ""}`}
          tabIndex={0}
          role="button"
          data-filter-id={id}
        >
          <div className="flex gap-2 items-center">
            {shortcut && (
              <KeyboardButton
                shortcut={shortcut}
                modifierKey="⌘"
                role="presentation"
                aria-haspopup="true"
                title={`Press '⌘${shortcut?.toUpperCase()}' to toggle ${label} options`}
              />
            )}
            <span className="text-[13px] text-accent-12 font-medium">{label}</span>
          </div>
          <div className="flex items-center gap-1.5">
            {filters.filter((filter) => filter.field === id).length > 0 && (
              <div className="bg-gray-6 rounded size-4 text-[11px] font-medium text-accent-12 text-center flex items-center justify-center">
                {filters.filter((filter) => filter.field === id).length}
              </div>
            )}
            <Button variant="ghost" size="icon" tabIndex={-1} className="size-5 [&_svg]:size-2">
              <CaretRight className="text-gray-7 group-hover:text-gray-10" />
            </Button>
          </div>
        </div>
      </PopoverTrigger>
      <PopoverContent
        className="w-60 bg-gray-1 dark:bg-black drop-shadow-2xl p-0 border-gray-6 rounded-lg"
        side="right"
        align="start"
        sideOffset={12}
        onKeyDown={handleKeyDown}
      >
        {component}
      </PopoverContent>
    </Popover>
  );
};
