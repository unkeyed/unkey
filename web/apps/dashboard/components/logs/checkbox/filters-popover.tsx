import { useKeyboardShortcut } from "@/hooks/use-keyboard-shortcut";
import { CaretRight, Check, Magnifier } from "@unkey/icons";
import { Drover, Input, KeyboardButton } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import React, {
  type KeyboardEvent,
  type PropsWithChildren,
  type Dispatch,
  type SetStateAction,
  useEffect,
  useRef,
  useState,
  useCallback,
  useMemo,
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

type FilterSearchItemBase = {
  id: string;
  label: string;
  path: string[];
  keywords?: string[];
  description?: string;
  icon?: React.ReactNode;
};

export type FilterSearchItem = FilterSearchItemBase &
  (
    | {
        kind: "filter";
        filterId: string;
      }
    | {
        kind: "option";
        checked: boolean;
        onSelect: () => void;
      }
  );

type FiltersPopoverProps = {
  items: FilterItemConfig[];
  activeFilters: FilterValue[];
  searchItems?: FilterSearchItem[];
  searchPlaceholder?: string;
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
  searchItems,
  searchPlaceholder = "Search filters and values...",
  open,
  onOpenChange,
  getFilterCount = (field) => activeFilters.filter((f) => f?.field === field).length,
}: PropsWithChildren<FiltersPopoverProps>) => {
  const [focusedIndex, setFocusedIndex] = useState<number | null>(null);
  const [activeFilter, setActiveFilter] = useState<string | null>(null);
  const [lastFocusedIndex, setLastFocusedIndex] = useState<number | null>(null);
  const [focusedSearchIndex, setFocusedSearchIndex] = useState<number | null>(null);
  const [search, setSearch] = useState("");
  const triggerRef = useRef<HTMLButtonElement | null>(null);

  const searchableItems = useMemo<FilterSearchItem[]>(
    () => [
      ...items.map(
        (item): FilterSearchItem => ({
          kind: "filter",
          id: `filter:${item.id}`,
          filterId: item.id,
          label: item.label,
          path: [],
        }),
      ),
      ...(searchItems ?? []),
    ],
    [items, searchItems],
  );

  const matchingSearchItems = useMemo(() => {
    const query = search.trim().toLocaleLowerCase();
    if (!query || !searchItems) {
      return [];
    }

    return searchableItems
      .filter((item) =>
        [item.label, item.description, ...item.path, ...(item.keywords ?? [])]
          .filter((value): value is string => value !== undefined)
          .some((value) => value.toLocaleLowerCase().includes(query)),
      )
      .slice(0, 50);
  }, [search, searchItems, searchableItems]);

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
      setFocusedSearchIndex(null);
      setSearch("");
    }
  }, [isOpen]);

  useEffect(() => {
    setFocusedSearchIndex(matchingSearchItems.length > 0 ? 0 : null);
  }, [matchingSearchItems.length]);

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

  const activateSearchItem = useCallback(
    (item: FilterSearchItem) => {
      if (item.kind === "filter") {
        setSearch("");
        handleActivateFilter(item.filterId);
      } else {
        item.onSelect();
      }
    },
    [handleActivateFilter],
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

    if (isInputFocused && e.key !== "Escape") {
      return;
    }

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
        className={cn(
          "min-w-60 bg-gray-1 dark:bg-black shadow-2xl border-gray-6 rounded-lg",
          searchItems ? "w-80 p-0" : "p-2",
        )}
        align="start"
        onKeyDown={handleKeyDown}
      >
        <div className="flex w-full flex-col">
          {searchItems ? (
            <div className="border-b border-gray-4 p-2">
              <Input
                autoFocus
                value={search}
                onChange={(event) => {
                  setSearch(event.target.value);
                  setFocusedSearchIndex(0);
                }}
                onKeyDown={(event) => {
                  if (event.key === "Escape") {
                    return;
                  }
                  event.stopPropagation();

                  if (matchingSearchItems.length === 0) {
                    return;
                  }
                  if (event.key === "ArrowDown") {
                    event.preventDefault();
                    setFocusedSearchIndex((current) =>
                      current === null ? 0 : (current + 1) % matchingSearchItems.length,
                    );
                  } else if (event.key === "ArrowUp") {
                    event.preventDefault();
                    setFocusedSearchIndex((current) =>
                      current === null
                        ? matchingSearchItems.length - 1
                        : (current - 1 + matchingSearchItems.length) % matchingSearchItems.length,
                    );
                  } else if (event.key === "Enter" && focusedSearchIndex !== null) {
                    event.preventDefault();
                    const item = matchingSearchItems[focusedSearchIndex];
                    if (item) {
                      activateSearchItem(item);
                    }
                  }
                }}
                placeholder={searchPlaceholder}
                aria-label="Search filters and values"
                variant="ghost"
                className="h-8 text-xs"
                leftIcon={<Magnifier className="size-3.5 text-gray-9" />}
                rightIcon={<KeyboardButton shortcut="F" />}
              />
            </div>
          ) : (
            <PopoverHeader />
          )}

          <div className={cn("w-full p-2", searchItems ? "max-h-80 overflow-y-auto" : "")}>
            {search.trim() && searchItems ? (
              matchingSearchItems.length > 0 ? (
                <div className="flex flex-col gap-1">
                  {matchingSearchItems.map((item, index) => (
                    <button
                      key={item.id}
                      type="button"
                      className={cn(
                        "flex h-9 w-full items-center gap-2 rounded-md px-2 text-left outline-hidden",
                        "hover:bg-gray-3 focus-visible:ring-2 focus-visible:ring-accent-7",
                        focusedSearchIndex === index && "bg-gray-3",
                        item.kind === "option" && item.checked && "bg-gray-3",
                      )}
                      aria-pressed={item.kind === "option" ? item.checked : undefined}
                      onMouseEnter={() => setFocusedSearchIndex(index)}
                      onClick={() => activateSearchItem(item)}
                    >
                      {item.kind === "option" ? (
                        <span
                          className={cn(
                            "flex size-4 shrink-0 items-center justify-center rounded-sm border border-gray-5",
                            item.checked &&
                              "border-accent-9 bg-accent-9 text-white dark:text-black",
                          )}
                          aria-hidden="true"
                        >
                          {item.checked ? <Check className="size-3" /> : null}
                        </span>
                      ) : null}
                      {item.icon ? (
                        <span className="shrink-0 text-accent-9">{item.icon}</span>
                      ) : null}
                      <span className="flex min-w-0 items-center gap-1 text-xs">
                        {item.path.map((segment) => (
                          <React.Fragment key={`${item.id}-${segment}`}>
                            <span className="max-w-28 truncate text-gray-9">{segment}</span>
                            <CaretRight className="size-2 shrink-0 text-gray-7" />
                          </React.Fragment>
                        ))}
                        <span className="truncate font-medium text-accent-12">{item.label}</span>
                      </span>
                      {item.description ? (
                        <span className="ml-auto max-w-28 shrink-0 truncate font-mono text-[10px] text-gray-8">
                          {item.description}
                        </span>
                      ) : null}
                      {item.kind === "filter" ? (
                        <CaretRight className="ml-auto size-2 shrink-0 text-gray-7" />
                      ) : null}
                    </button>
                  ))}
                </div>
              ) : (
                <div className="px-2 py-8 text-center text-xs text-gray-9">No filters found</div>
              )
            ) : (
              <div className="flex flex-col gap-2 w-full" role="menu">
                {items.map((item, index) => (
                  <FilterItem
                    key={item.id}
                    {...item}
                    filterCount={getFilterCount(item.id)}
                    isFocused={focusedIndex === index}
                    isActive={activeFilter === item.id}
                    setActiveFilter={setActiveFilter}
                  />
                ))}
              </div>
            )}
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
