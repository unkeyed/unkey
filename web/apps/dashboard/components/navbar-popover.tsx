"use client";
import { DisabledWrapper } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/components/disabled-wrapper";
import { useKeyboardShortcut } from "@/hooks/use-keyboard-shortcut";
import { useVirtualizer } from "@tanstack/react-virtual";
import { CaretRight } from "@unkey/icons";
import { KeyboardButton, Popover, PopoverContent, PopoverTrigger } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import { usePathname, useRouter } from "next/navigation";
import { type PropsWithChildren, useCallback, useEffect, useRef, useState } from "react";

export type QuickNavItem = {
  id: string;
  label: React.ReactNode;
  shortcut?: string;
  href?: string;
  onClick?: () => void;
  className?: string;
  itemClassName?: string;
  hideRightIcon?: boolean;
  disabled?: boolean;
  disabledTooltip?: string;
};

type QuickNavPopoverProps = {
  items: QuickNavItem[];
  title?: string;
  shortcutKey?: string;
  onItemSelect?: (item: QuickNavItem) => void;
  /**
   * Explicitly specify which item should be highlighted as active.
   * Takes precedence over path-based matching.
   */
  activeItemId?: string;
  /**
   * Threshold for when to use virtualization.
   * Lists with fewer items than this will render without virtualization.
   * @default 10
   */
  virtualizationThreshold?: number;
};

export const QuickNavPopover = ({
  children,
  items,
  title = "Navigate to...",
  shortcutKey = "f",
  onItemSelect,
  activeItemId,
  virtualizationThreshold = 10,
}: PropsWithChildren<QuickNavPopoverProps>) => {
  const pathname = usePathname();
  const router = useRouter();
  const [open, setOpen] = useState(false);
  const [focusedIndex, setFocusedIndex] = useState<number | null>(null);
  const listRef = useRef<HTMLDivElement>(null);

  // Determine if we should use virtualization
  const useVirtual = items.length > virtualizationThreshold;

  // Calculate max height based on item count, but capped
  // For few items, calculate exact height needed (36px per item)
  const exactHeight = items.length * 36;
  const maxHeight = Math.min(exactHeight, 200);

  // Only apply scrolling if items exceed threshold
  const shouldScroll = items.length > 6;

  // Always call useVirtualizer unconditionally (Rules of Hooks)
  // We'll use the useVirtual boolean to determine whether to use its result
  const rowVirtualizer = useVirtualizer({
    count: items.length,
    getScrollElement: () => listRef.current,
    estimateSize: () => 32,
    overscan: 5,
  });

  useKeyboardShortcut(`option+shift+${shortcutKey}`, () => {
    setOpen((prev) => !prev);
  });

  const handleItemSelect = (item: QuickNavItem) => {
    if (item.disabled) {
      return;
    }
    setOpen(false);
    if (onItemSelect) {
      onItemSelect(item);
      return;
    }
    if (item.onClick) {
      item.onClick();
      return;
    }
    if (item.href) {
      router.push(item.href);
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (!open) {
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
          const selectedItem = items[focusedIndex];
          handleItemSelect(selectedItem);
        }
        break;
    }
  };

  // Find the active item index for initial focus
  const findActiveItemIndex = useCallback(() => {
    // Explicit activeItemId takes precedence
    if (activeItemId) {
      const activeIndex = items.findIndex((item) => item.id === activeItemId);
      return activeIndex >= 0 ? activeIndex : 0;
    }

    // Fall back to path-based matching
    if (!pathname) {
      return 0;
    }

    const activeIndex = items.findIndex((item) => item.href && checkIsActive(item.href, pathname));

    return activeIndex >= 0 ? activeIndex : 0;
  }, [items, pathname, activeItemId]);

  // Scroll to focused item when using virtualization
  useEffect(() => {
    if (focusedIndex !== null && useVirtual && rowVirtualizer) {
      rowVirtualizer.scrollToIndex(focusedIndex, { align: "auto" });
    }
  }, [focusedIndex, useVirtual, rowVirtualizer]);

  // Set initial focus when opening popover
  useEffect(() => {
    if (open) {
      setFocusedIndex(findActiveItemIndex());
    } else {
      setFocusedIndex(null);
    }
  }, [open, findActiveItemIndex]);

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger>{children}</PopoverTrigger>
      <PopoverContent
        className={cn(
          "w-60 bg-gray-1 dark:bg-black drop-shadow-2xl transform-gpu p-2 border-gray-6 rounded-lg",
        )}
        align="start"
        onKeyDown={handleKeyDown}
      >
        <div className="flex flex-col gap-2">
          <PopoverHeader title={title} shortcutKey={shortcutKey.toUpperCase()} />

          {/* Container for list items */}
          <div
            ref={listRef}
            className={cn(
              "w-full",
              shouldScroll ? "overflow-auto" : "overflow-visible",
              useVirtual ? "" : "flex flex-col gap-1",
            )}
            style={{
              height: useVirtual ? maxHeight : shouldScroll ? maxHeight : "auto",
              maxHeight: maxHeight,
              width: "100%",
            }}
          >
            {useVirtual ? (
              <div
                className="relative w-full"
                style={{
                  height: `${rowVirtualizer?.getTotalSize()}px`,
                }}
              >
                {rowVirtualizer?.getVirtualItems().map((virtualRow) => {
                  const item = items[virtualRow.index];
                  const isActive = isItemActive(item, pathname, activeItemId);

                  return (
                    <div
                      key={virtualRow.index}
                      data-index={virtualRow.index}
                      className="absolute top-0 left-0 w-full"
                      style={{
                        height: `${virtualRow.size}px`,
                        transform: `translateY(${virtualRow.start}px)`,
                      }}
                    >
                      <PopoverItem
                        {...item}
                        isActive={isActive}
                        isFocused={focusedIndex === virtualRow.index}
                        onSelect={() => handleItemSelect(item)}
                      />
                    </div>
                  );
                })}
              </div>
            ) : (
              items.map((item, index) => {
                const isActive = isItemActive(item, pathname, activeItemId);

                return (
                  <PopoverItem
                    isActive={isActive}
                    key={item.id}
                    {...item}
                    isFocused={focusedIndex === index}
                    onSelect={() => handleItemSelect(item)}
                  />
                );
              })
            )}
          </div>
        </div>
      </PopoverContent>
    </Popover>
  );
};

const PopoverHeader = ({
  title,
  shortcutKey,
}: {
  title: string;
  shortcutKey: string;
}) => (
  <div className="flex w-full justify-between items-center px-2 py-1">
    <span className="text-gray-9 text-[13px]">{title}</span>
    <KeyboardButton shortcut={`⌥+⇧+${shortcutKey}`} />
  </div>
);

type PopoverItemProps = QuickNavItem & {
  isFocused?: boolean;
  isActive?: boolean;
  onSelect: () => void;
};

const PopoverItem = ({
  label,
  isFocused,
  isActive,
  onSelect,
  className,
  itemClassName,
  hideRightIcon,
  disabled,
  disabledTooltip,
}: PopoverItemProps) => {
  const itemRef = useRef<HTMLButtonElement>(null);

  useEffect(() => {
    if (isFocused && itemRef.current) {
      itemRef.current.focus();
    }
  }, [isFocused]);
  const labelText = typeof label === "string" ? label : "";

  const content = (
    <DisabledWrapper
      disabled={Boolean(disabled)}
      tooltipContent={disabledTooltip}
      className="py-0 px-0 pointer-events-none cursor"
    >
      <button
        type="button"
        ref={itemRef}
        className={cn(
          "flex w-full items-center px-2 py-1.5 justify-between rounded-lg group cursor-pointer",
          "hover:bg-gray-3 data-[state=open]:bg-gray-3 focus:outline-none",
          (isFocused || isActive) && "bg-gray-3",
          itemClassName,
        )}
        tabIndex={0}
        onClick={onSelect}
      >
        <div className={cn("flex gap-2 items-center", className)}>
          <span
            className={"text-[13px] font-medium truncate max-w-[160px] text-accent-12"}
            title={labelText}
          >
            {label}
          </span>
        </div>
        {!hideRightIcon && (
          <div className="flex items-center gap-1.5">
            <div className="size-5 flex items-center justify-center">
              <CaretRight
                className={cn(
                  "size-2",
                  isActive ? "text-gray-12" : "text-gray-7 group-hover:text-gray-10",
                )}
              />
            </div>
          </div>
        )}
      </button>
    </DisabledWrapper>
  );

  return content;
};

const isItemActive = (
  item: QuickNavItem,
  pathname: string | null,
  activeItemId?: string,
): boolean => {
  if (activeItemId !== undefined) {
    return item.id === activeItemId;
  }
  return Boolean(item.href && checkIsActive(item.href, pathname));
};

const checkIsActive = (itemPath: string, currentPath: string | null) => {
  if (!itemPath) {
    return false;
  }

  // Normalize paths by removing trailing slashes
  const normalizedItemPath = itemPath.endsWith("/") ? itemPath.slice(0, -1) : itemPath;
  const normalizedCurrentPath = currentPath?.endsWith("/") ? currentPath.slice(0, -1) : currentPath;

  // Exact match check
  if (normalizedItemPath === normalizedCurrentPath) {
    return true;
  }

  return false;
};
