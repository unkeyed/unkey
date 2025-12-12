import { useKeyboardShortcut } from "@/hooks/use-keyboard-shortcut";
import { Drover, KeyboardButton } from "@unkey/ui";
import React, {
  type KeyboardEvent,
  type PropsWithChildren,
  type Dispatch,
  type SetStateAction,
  useEffect,
  useRef,
  useState,
  useCallback,
} from "react";
import type { FilterValue } from "../validation/filter.types";
import { FilterItem } from "./filter-item";

export type FilterItemConfig = {
  id: string;
  label: string;
  shortcut?: string;
  shortcutLabel?: string;
  component: React.ReactNode;
};

type FiltersPopoverProps = {
  items: FilterItemConfig[];
  activeFilters: FilterValue[];
  getFilterCount?: (field: string) => number;
  open?: boolean;
  onOpenChange?: Dispatch<SetStateAction<boolean>>;
};

// INFO: Workaround for applying hooks dynamically: Render a separate (null)
// ShortcutActivator component for each item's shortcut below. This allows
// top-level 'useKeyboardShortcut' calls per item, avoiding manual listener boilerplate,
// even if the component structure feels a bit indirect ("hacky").
const ShortcutActivator = React.memo(
  ({
    shortcut,
    id,
    onActivate,
  }: {
    shortcut: string;
    id: string;
    onActivate: (id: string) => void;
  }) => {
    useKeyboardShortcut(shortcut, () => onActivate(id), {
      preventDefault: true,
      ignoreInputs: true,
      ignoreContentEditable: true,
    });
    return null; // Render nothing
  },
);
ShortcutActivator.displayName = "ShortcutActivator";

export const FiltersPopover = ({
  children,
  items = [],
  activeFilters = [],
  open,
  onOpenChange,
  getFilterCount = (field) => activeFilters.filter((f) => f?.field === field).length,
}: PropsWithChildren<FiltersPopoverProps>) => {
  const [focusedIndex, setFocusedIndex] = useState<number | null>(null);
  const [activeFilter, setActiveFilter] = useState<string | null>(null);
  const [lastFocusedIndex, setLastFocusedIndex] = useState<number | null>(null);
  const triggerRef = useRef<HTMLButtonElement | null>(null);

  // Handle local state if external state isn't provided
  const [internalOpen, setInternalOpen] = useState(false);
  const isControlled = open !== undefined && onOpenChange !== undefined;
  const isOpen = isControlled ? open : internalOpen;
  const setOpen = useCallback(
    (value: boolean | ((prev: boolean) => boolean)) => {
      if (isControlled) {
        const nextValue = typeof value === "function" ? value(!!open) : value;
        onOpenChange?.(nextValue);
      } else {
        setInternalOpen(value);
      }
    },
    [isControlled, open, onOpenChange],
  );

  useEffect(() => {
    if (!isOpen) {
      setActiveFilter(null);
      setFocusedIndex(null);
      setLastFocusedIndex(null);
    }
  }, [isOpen]);

  useEffect(() => {
    if (!activeFilter && lastFocusedIndex !== null && isOpen) {
      setFocusedIndex(lastFocusedIndex);
    }
  }, [activeFilter, lastFocusedIndex, isOpen]);

  useKeyboardShortcut(
    "f",
    () => {
      setOpen((prev) => {
        const newState = !prev;
        if (newState && items.length > 0) {
          setTimeout(() => setFocusedIndex(0), 0);
        }
        return newState;
      });
    },
    { preventDefault: true, ignoreInputs: true },
  );

  const handleActivateFilter = useCallback(
    (id: string) => {
      setOpen(true);
      setTimeout(() => {
        setActiveFilter(id);
        const index = items.findIndex((i) => i.id === id);
        if (index !== -1) {
          setFocusedIndex(index);
          setLastFocusedIndex(index);
        }
      }, 0);
    },
    [items, setOpen],
  );

  const handleKeyDown = (e: KeyboardEvent) => {
    if (!isOpen) {
      return;
    }

    const targetElement = e.target as HTMLElement;
    const isInputFocused =
      targetElement.tagName === "INPUT" ||
      targetElement.tagName === "TEXTAREA" ||
      targetElement.isContentEditable;

    // If a filter item popover is active, only handle ArrowLeft (outside inputs)
    if (activeFilter) {
      if (e.key === "ArrowLeft" && !isInputFocused) {
        e.preventDefault();
        const closingIndex = items.findIndex((i) => i.id === activeFilter);
        if (closingIndex !== -1) {
          setLastFocusedIndex(closingIndex); // Remember index to return focus to
        }
        setActiveFilter(null); // Deactivate child popover
        // useEffect [activeFilter] will handle setting focusedIndex based on lastFocusedIndex
      }
      // Stop parent handling other keys when child is active
      return;
    }

    // Handle navigation in the main filter list (when activeFilter is null)
    switch (e.key) {
      case "ArrowDown": {
        e.preventDefault();
        const newIndex = focusedIndex === null ? 0 : (focusedIndex + 1) % items.length;
        setFocusedIndex(newIndex);
        setLastFocusedIndex(newIndex); // Keep track for potential activation
        break;
      }
      case "ArrowUp": {
        e.preventDefault();
        const newIndex =
          focusedIndex === null
            ? items.length - 1
            : (focusedIndex - 1 + items.length) % items.length;
        setFocusedIndex(newIndex);
        setLastFocusedIndex(newIndex); // Keep track
        break;
      }
      case "Enter":
      case "ArrowRight": {
        e.preventDefault();
        if (focusedIndex !== null) {
          const selectedFilter = items[focusedIndex];
          if (selectedFilter) {
            setLastFocusedIndex(focusedIndex); // Store index before activating
            setActiveFilter(selectedFilter.id); // Activate the child popover
          }
        }
        break;
      }
      case "Escape": {
        e.preventDefault();
        setOpen(false); // Close the main popover
        break;
      }
    }
  };

  return (
    <Drover.Root open={isOpen} onOpenChange={setOpen}>
      {/* Render Shortcut Activators (these components render null) */}
      {/* These must be rendered for the hooks inside them to be active */}
      {items.map((item) =>
        item.shortcut ? (
          <ShortcutActivator
            key={`${item.id}-shortcut`} // Unique key for the activator
            shortcut={item.shortcut}
            id={item.id}
            onActivate={handleActivateFilter}
          />
        ) : null,
      )}

      <Drover.Trigger asChild ref={triggerRef}>
        {children}
      </Drover.Trigger>

      <Drover.Content
        className="min-w-60 bg-gray-1 dark:bg-black shadow-2xl p-2 border-gray-6 rounded-lg"
        align="start"
        onKeyDown={handleKeyDown}
      >
        <div className="flex flex-col gap-2 w-full">
          <PopoverHeader />
          <div className="flex flex-col gap-2 w-full" role="menu">
            {items.map((item, index) => (
              <FilterItem
                key={item.id}
                {...item} // Pass item config props (id, label, component, shortcut for display)
                filterCount={getFilterCount(item.id)}
                isFocused={focusedIndex === index} // Is this item highlighted in the list?
                isActive={activeFilter === item.id} // Is this item's popover open?
                setActiveFilter={setActiveFilter}
              />
            ))}
          </div>
        </div>
      </Drover.Content>
    </Drover.Root>
  );
};

const PopoverHeader = () => (
  <div className="flex w-full justify-between items-center px-2 py-1">
    <span className="text-gray-9 text-[13px]">Filters...</span>
    <KeyboardButton shortcut="F" />
  </div>
);
