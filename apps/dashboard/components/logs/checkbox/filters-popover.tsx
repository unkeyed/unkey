import { KeyboardButton } from "@/components/keyboard-button";
import { Drover } from "@/components/ui/drover";
import { useKeyboardShortcut } from "@/hooks/use-keyboard-shortcut";
import { CaretRight } from "@unkey/icons";
import { Button } from "@unkey/ui";
import {
  type Dispatch,
  type KeyboardEvent,
  type PropsWithChildren,
  type SetStateAction,
  useEffect,
  useRef,
  useState,
} from "react";
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
  open?: boolean;
  onOpenChange?: Dispatch<SetStateAction<boolean>>;
};

export const FiltersPopover = ({
  children,
  items,
  activeFilters,
  open,
  onOpenChange,
  getFilterCount = (field) => activeFilters.filter((f) => f.field === field).length,
}: PropsWithChildren<FiltersPopoverProps>) => {
  const [focusedIndex, setFocusedIndex] = useState<number | null>(null);
  const [activeFilter, setActiveFilter] = useState<string | null>(null);

  // biome-ignore lint/correctness/useExhaustiveDependencies: no need
  useEffect(() => {
    return () => setActiveFilter(null);
  }, [open]);

  useKeyboardShortcut("f", () => {
    onOpenChange?.((prev) => !prev);
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
    <Drover.Root open={open} onOpenChange={onOpenChange}>
      <Drover.Trigger asChild>{children}</Drover.Trigger>
      <Drover.Content onKeyDown={handleKeyDown}>
        <div className="flex flex-col w-full">
          <DroverHeader />
          <div className="p-2">
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
        </div>
      </Drover.Content>
    </Drover.Root>
  );
};

const DroverHeader = () => (
  <div className="flex w-full justify-between items-center px-4 pt-3 md:px-3 md:py-1">
    <span className="text-gray-9 text-sm">Filters</span>
    <KeyboardButton shortcut="F" className="max-md:hidden" />
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
    <Drover.Nested>
      <Drover.Trigger asChild>
        <div
          ref={itemRef}
          className={`flex w-full items-center px-2 py-1.5 justify-between rounded-lg group cursor-pointer
          hover:bg-gray-3 data-[state=open]:bg-gray-3 focus:outline-none
          ${isFocused ? "bg-gray-3" : ""}`}
          tabIndex={0}
          // biome-ignore lint/a11y/useSemanticElements: its okay
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
      </Drover.Trigger>
      <Drover.Content
        ref={contentRef}
        className="min-w-60 z-50 w-full bg-gray-1 dark:bg-black drop-shadow-2xl p-0 border-gray-6 rounded-lg"
        side="right"
        align="start"
        sideOffset={12}
        onKeyDown={handleKeyDown}
        tabIndex={-1}
      >
        {component}
      </Drover.Content>
    </Drover.Nested>
  );
};
