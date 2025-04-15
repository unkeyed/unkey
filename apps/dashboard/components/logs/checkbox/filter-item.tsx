import { KeyboardButton } from "@/components/keyboard-button";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { CaretRight } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import type React from "react";
import { type KeyboardEvent, useCallback, useEffect, useRef, useState } from "react";

export type FilterItemConfig = {
  id: string;
  label: string;
  shortcut?: string;
  shortcutLabel?: string;
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
  shortcutLabel,
  component,
  isFocused,
  isActive,
  filterCount,
  setActiveFilter,
}: FilterItemProps) => {
  // Internal open state, primarily controlled by 'isActive' prop effect
  const [open, setOpen] = useState(isActive ?? false);
  const itemRef = useRef<HTMLDivElement>(null); // Ref for the trigger div
  const contentRef = useRef<HTMLDivElement>(null); // Ref for the PopoverContent

  // Synchronize internal open state with the parent's isActive prop
  useEffect(() => {
    setOpen(isActive ?? false);
  }, [isActive]);

  // Focus the trigger div when parent indicates it's focused in the main list
  // biome-ignore lint/correctness/useExhaustiveDependencies: <explanation>
  useEffect(() => {
    if (isFocused && !isActive && itemRef.current) {
      // Only focus trigger if not active
      // console.log(`Focusing trigger for: ${label}`);
      itemRef.current.focus({ preventScroll: true });
    }
  }, [isFocused, isActive, label]); // Depend on isActive too

  // Focus content when popover becomes active and open
  // biome-ignore lint/correctness/useExhaustiveDependencies: <explanation>
  useEffect(() => {
    if (isActive && open && contentRef.current) {
      // console.log(`Attempting to focus content for: ${label}`);
      // Find and focus the first focusable element within the content
      const focusableElements = contentRef.current.querySelectorAll<HTMLElement>(
        'button, [href], input:not([type="hidden"]), select, textarea, [tabindex]:not([tabindex="-1"])',
      );
      if (focusableElements.length > 0) {
        // console.log(`Focusing first element:`, focusableElements[0]);
        focusableElements[0].focus({ preventScroll: true });
      } else {
        // Fallback: focus the content container itself if nothing else is focusable
        contentRef.current.focus({ preventScroll: true });
      }
    }
  }, [isActive, open, label]); // Depend on isActive and open

  // --- Removed redundant useKeyboardShortcut ---
  // Activation is handled by the parent component (FiltersPopover)

  // --- Event Handlers ---

  // KeyDown handler for the PopoverContent (specific to this item's popover)
  const handleItemPopoverKeyDown = useCallback(
    (e: KeyboardEvent) => {
      // No need to check isInputFocused here as parent handles ArrowLeft navigation back
      // We only care about Escape to close *this* popover.
      if (e.key === "Escape") {
        e.preventDefault();
        e.stopPropagation(); // Stop Escape from bubbling further (e.g., closing main popover)
        // Request parent to deactivate this filter and focus the trigger
        setActiveFilter(null);
        // Focus should return naturally because parent will set isFocused=true
        // based on lastFocusedIndex after setActiveFilter(null) is processed.
        // Explicitly focusing here might cause race conditions.
        // itemRef.current?.focus(); // Avoid focusing immediately here
      }
      // Allow other keys (like arrows in inputs) to behave normally
    },
    [setActiveFilter], // Depend on the callback from parent
  );

  // Handler for Popover's open state changes (e.g., clicking outside)
  // biome-ignore lint/correctness/useExhaustiveDependencies: <explanation>
  const handleOpenChange = useCallback(
    (newOpenState: boolean) => {
      // This function is called by Radix when the popover intends to close
      // (e.g., click outside, Escape press handled by Radix internally if not stopped)
      setOpen(newOpenState); // Keep internal state synced

      // If Radix closed the popover AND the parent still thinks it's active,
      // we MUST inform the parent to update its state.
      if (!newOpenState && isActive) {
        // console.log(`Popover for ${label} closed via interaction.`);
        setActiveFilter(null);
      }
      // If it opened via interaction (shouldn't happen if controlled),
      // or closed when parent already knew, do nothing extra.
    },
    [isActive, setActiveFilter, label],
  );

  // Handler for clicking the trigger element
  const handleTriggerClick = useCallback(() => {
    // Toggle activation by telling the parent
    setActiveFilter(isActive ? null : id);
  }, [isActive, id, setActiveFilter]);

  // --- Render ---
  return (
    <Popover open={open} onOpenChange={handleOpenChange}>
      <PopoverTrigger asChild>
        {/* biome-ignore lint/a11y/useKeyWithClickEvents: <explanation> */}
        <div
          ref={itemRef}
          className={cn(
            "flex w-full items-center px-2 py-1.5 justify-between rounded-lg group cursor-pointer",
            "hover:bg-gray-3 data-[state=open]:bg-gray-3",
            "focus:outline-none",
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
            {" "}
            {/* Prevent inner elements catching click */}
            {shortcut && (
              <KeyboardButton
                shortcut={shortcutLabel ?? shortcut}
                // modifierKey={null} // Omit modifierKey prop
                role="presentation"
                aria-hidden="true"
                // Simple title using the shortcut string
                title={`Shortcut: ${shortcut}`}
              />
            )}
            <span className="text-[13px] text-accent-12 font-medium select-none">{label}</span>
          </div>
          <div className="flex items-center gap-1.5 pointer-events-none">
            {filterCount > 0 && (
              <div className="bg-gray-6 rounded size-4 text-[11px] font-medium text-accent-12 text-center flex items-center justify-center">
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
      </PopoverTrigger>
      <PopoverContent
        ref={contentRef}
        className="min-w-60 w-full bg-gray-1 dark:bg-black drop-shadow-2xl p-0 border-gray-6 rounded-lg outline-none"
        side="right"
        align="start"
        sideOffset={8}
        onKeyDown={handleItemPopoverKeyDown} // Handle Escape within content
        tabIndex={-1} // Make content container focusable for fallback
      >
        {component}
      </PopoverContent>
    </Popover>
  );
};
