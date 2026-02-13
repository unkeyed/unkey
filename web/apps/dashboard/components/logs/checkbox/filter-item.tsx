import { CaretRight } from "@unkey/icons";
import { Button, Drover, KeyboardButton } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import type React from "react";
import { type KeyboardEvent, useCallback, useEffect, useRef, useState } from "react";

export type FilterItemConfig = {
  id: string;
  label: string;
  shortcut?: string;
  component: React.ReactNode;
};

type FilterItemProps = FilterItemConfig & {
  isFocused?: boolean; // Highlighted in the main list?
  isActive?: boolean; // Is this item's popover the active one?
  filterCount: number;
  setActiveFilter: (id: string | null) => void;
};

export const FilterItem = ({
  id,
  label,
  shortcut,
  component,
  isFocused,
  isActive,
  filterCount,
  setActiveFilter,
}: FilterItemProps) => {
  // Internal open state, primarily controlled by 'isActive' prop effect
  const [open, setOpen] = useState(isActive ?? false);
  const itemRef = useRef<HTMLDivElement>(null); // Ref for the trigger div
  const contentRef = useRef<HTMLDivElement>(null); // Ref for the DroverContent

  // Synchronize internal open state with the parent's isActive prop
  useEffect(() => {
    setOpen(isActive ?? false);
  }, [isActive]);

  // Focus the trigger div when parent indicates it's focused in the main list
  // biome-ignore lint/correctness/useExhaustiveDependencies:  no need to react for label
  useEffect(() => {
    if (isFocused && !isActive && itemRef.current) {
      // Only focus trigger if not active
      itemRef.current.focus({ preventScroll: true });
    }
  }, [isFocused, isActive, label]); // Depend on isActive too

  // Focus content when drover becomes active and open
  // biome-ignore lint/correctness/useExhaustiveDependencies:  no need to react for label
  useEffect(() => {
    if (isActive && open && contentRef.current) {
      // Find and focus the first focusable element within the content
      const focusableElements = contentRef.current.querySelectorAll<HTMLElement>(
        'button, [href], input:not([type="hidden"]), select, textarea, [tabindex]:not([tabindex="-1"])',
      );
      if (focusableElements.length > 0) {
        focusableElements[0].focus({ preventScroll: true });
      } else {
        // Fallback: focus the content container itself if nothing else is focusable
        contentRef.current.focus({ preventScroll: true });
      }
    }
  }, [isActive, open, label]); // Depend on isActive and open

  const handleItemDroverKeyDown = useCallback(
    (e: KeyboardEvent) => {
      // No need to check isInputFocused here as parent handles ArrowLeft navigation back
      // We only care about Escape to close *this* drover.
      if (e.key === "Escape") {
        e.preventDefault();
        e.stopPropagation(); // Stop Escape from bubbling further (e.g., closing main drover)
        // Request parent to deactivate this filter and focus the trigger
        setActiveFilter(null);
        // Focus should return naturally because parent will set isFocused=true
        // based on lastFocusedIndex after setActiveFilter(null) is processed.
      }
      // Allow other keys (like arrows in inputs) to behave normally
    },
    [setActiveFilter], // Depend on the callback from parent
  );

  // Handler for Drover's open state changes (e.g., clicking outside)
  const handleOpenChange = useCallback(
    (newOpenState: boolean) => {
      // This function is called when the drover intends to close
      // (e.g., click outside, Escape press handled internally if not stopped)
      setOpen(newOpenState); // Keep internal state synced

      // If the drover closed AND the parent still thinks it's active,
      // we MUST inform the parent to update its state.
      if (!newOpenState && isActive) {
        setActiveFilter(null);
      }
      // If it opened via interaction (shouldn't happen if controlled),
      // or closed when parent already knew, do nothing extra.
    },
    [isActive, setActiveFilter],
  );

  // Handler for clicking the trigger element
  const handleTriggerClick = useCallback(() => {
    // Toggle activation by telling the parent
    setActiveFilter(isActive ? null : id);
  }, [isActive, id, setActiveFilter]);

  return (
    <Drover.Nested open={open} onOpenChange={handleOpenChange}>
      <Drover.Trigger asChild>
        {/* biome-ignore lint/a11y/useKeyWithClickEvents: <explanation> */}
        <div
          ref={itemRef}
          className={cn(
            "flex w-full items-center px-2 py-1.5 justify-between rounded-lg group cursor-pointer",
            "hover:bg-gray-3 data-[state=open]:bg-gray-3",
            "focus:outline-hidden focus:ring-2 focus:ring-accent-7",
            isFocused && !isActive ? "bg-gray-4" : "",
            isActive ? "bg-gray-3" : "",
          )}
          tabIndex={-1}
          role="menuitem"
          aria-haspopup="true"
          aria-expanded={open}
          onClick={handleTriggerClick}
        >
          <div className="flex gap-2 items-center pointer-events-none">
            {shortcut && (
              <KeyboardButton
                shortcut={shortcut}
                role="presentation"
                aria-haspopup="true"
                title={`Press '${shortcut?.toUpperCase()}' to toggle ${label} options`}
              />
            )}
            <span className="text-[13px] text-accent-12 font-medium select-none">{label}</span>
          </div>
          <div className="flex items-center gap-1.5 pointer-events-none">
            {filterCount > 0 && (
              <div className="bg-gray-6 rounded-sm size-4 text-[11px] font-medium text-accent-12 text-center flex items-center justify-center">
                {filterCount}
              </div>
            )}
            <Button
              variant="ghost"
              size="icon"
              tabIndex={-1} // Non-interactive button
              className="size-5 [&_svg]:size-2"
              aria-hidden="true"
            >
              <CaretRight className="text-gray-7 group-hover:text-gray-10" />
            </Button>
          </div>
        </div>
      </Drover.Trigger>
      <Drover.Content
        ref={contentRef}
        className="min-w-60 w-full bg-gray-1 dark:bg-black drop-shadow-2xl transform-gpu p-0 border-gray-6 rounded-lg"
        side="right"
        align="start"
        sideOffset={12}
        onKeyDown={handleItemDroverKeyDown}
        tabIndex={-1}
      >
        {component}
      </Drover.Content>
    </Drover.Nested>
  );
};
