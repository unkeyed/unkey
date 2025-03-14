import { KeyboardButton } from "@/components/keyboard-button";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { useKeyboardShortcut } from "@/hooks/use-keyboard-shortcut";
import { CaretRight } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { type KeyboardEvent, type PropsWithChildren, useEffect, useRef, useState } from "react";
import type { FilterValue } from "../validation/filter.types";

export type FilterItemConfig = {
  id: string;
  label: string;
  shortcut?: string;
  component: React.ReactNode;
};

type FiltersPopoverProps = {
  items: FilterItemConfig[];
  activeFilters: FilterValue[];
  getFilterCount?: (field: string) => number;
};

export const FiltersPopover = ({
  children,
  items,
  activeFilters,
  getFilterCount = (field) => activeFilters.filter((f) => f.field === field).length,
}: PropsWithChildren<FiltersPopoverProps>) => {
  const [open, setOpen] = useState(false);
  const [focusedIndex, setFocusedIndex] = useState<number | null>(null);
  const [activeFilter, setActiveFilter] = useState<string | null>(null);

  // biome-ignore lint/correctness/useExhaustiveDependencies: no need
  useEffect(() => {
    return () => setActiveFilter(null);
  }, [open]);

  useKeyboardShortcut("f", () => {
    setOpen((prev) => !prev);
  });

  const handleKeyDown = (e: KeyboardEvent) => {
    if (!open) {
      return;
    }

    if (activeFilter) {
      if (e.key === "ArrowLeft") {
        e.preventDefault();
        setActiveFilter(null);
      }
      return;
    }

    switch (e.key) {
      case "ArrowDown":
        e.preventDefault();
        setFocusedIndex((prev) => (prev === null ? 0 : (prev + 1) % items.length));
        break;
      case "ArrowUp":
        e.preventDefault();
        setFocusedIndex((prev) =>
          prev === null ? items.length - 1 : (prev - 1 + items.length) % items.length,
        );
        break;
      case "Enter":
      case "ArrowRight":
        e.preventDefault();
        if (focusedIndex !== null) {
          const selectedFilter = items[focusedIndex];
          if (selectedFilter) {
            setActiveFilter(selectedFilter.id);
          }
        }
        break;
    }
  };

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>{children}</PopoverTrigger>
      <PopoverContent
        className="min-w-60 bg-gray-1 dark:bg-black shadow-2xl p-2 border-gray-6 rounded-lg"
        align="start"
        onKeyDown={handleKeyDown}
      >
        <div className="flex flex-col gap-2 w-full">
          <PopoverHeader />
          {items.map((item, index) => (
            <FilterItem
              key={item.id}
              {...item}
              filterCount={getFilterCount(item.id)}
              isFocused={focusedIndex === index}
              isActive={activeFilter === item.id}
            />
          ))}
        </div>
      </PopoverContent>
    </Popover>
  );
};

const PopoverHeader = () => (
  <div className="flex w-full justify-between items-center px-2 py-1">
    <span className="text-gray-9 text-[13px]">Filters...</span>
    <KeyboardButton shortcut="F" />
  </div>
);

type FilterItemProps = FilterItemConfig & {
  isFocused?: boolean;
  isActive?: boolean;
  filterCount: number;
};

const FilterItem = ({
  label,
  shortcut,
  component,
  isFocused,
  isActive,
  filterCount,
}: FilterItemProps) => {
  const [open, setOpen] = useState(false);
  const itemRef = useRef<HTMLDivElement>(null);
  const contentRef = useRef<HTMLDivElement>(null);

  const handleKeyDown = (e: KeyboardEvent) => {
    if (e.key === "ArrowLeft" && open) {
      e.preventDefault();
      setOpen(false);
      itemRef.current?.focus();
    }
  };

  useKeyboardShortcut({ key: shortcut || "", meta: true }, () => setOpen(true), {
    preventDefault: true,
  });

  useEffect(() => {
    if (isFocused && itemRef.current) {
      itemRef.current.focus();
    }
  }, [isFocused]);

  useEffect(() => {
    if (isActive && !open) {
      setOpen(true);
    }
    if (isActive && open && contentRef.current) {
      const focusableElements = contentRef.current.querySelectorAll(
        'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])',
      );
      if (focusableElements.length > 0) {
        (focusableElements[0] as HTMLElement).focus();
      } else {
        contentRef.current.focus();
      }
    }
  }, [isActive, open]);

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
            {filterCount > 0 && (
              <div className="bg-gray-6 rounded size-4 text-[11px] font-medium text-accent-12 text-center flex items-center justify-center">
                {filterCount}
              </div>
            )}
            <Button variant="ghost" size="icon" tabIndex={-1} className="size-5 [&_svg]:size-2">
              <CaretRight className="text-gray-7 group-hover:text-gray-10" />
            </Button>
          </div>
        </div>
      </PopoverTrigger>
      <PopoverContent
        ref={contentRef}
        className="min-w-60 w-full bg-gray-1 dark:bg-black drop-shadow-2xl p-0 border-gray-6 rounded-lg"
        side="right"
        align="start"
        sideOffset={12}
        onKeyDown={handleKeyDown}
        tabIndex={-1}
      >
        {component}
      </PopoverContent>
    </Popover>
  );
};
