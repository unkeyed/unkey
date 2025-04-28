"use client";
import { Dialog, DialogContent } from "@/components/ui/dialog";
import type { IconProps } from "@unkey/icons/src/props";
import { Button } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import { createContext, useCallback, useContext, useEffect, useState } from "react";
import type { FC, ReactNode } from "react";
import { DefaultDialogContentArea, DefaultDialogFooter, DefaultDialogHeader } from "./dialog-parts";

type NavigableDialogContextType<TStepName extends string> = {
  activeId: TStepName | undefined;
  setActiveId: (id: TStepName) => void;
};

const createNavigableDialogContext = <TStepName extends string>() => {
  return createContext<NavigableDialogContextType<TStepName> | undefined>(undefined);
};

// We're using a type assertion with 'any' here as a deliberate design choice.
// The actual type safety is enforced later by the typed hooks
// (useNavigableDialog<T>) that properly cast this context to the correct generic type.
// @ts-expect-error Type 'Context<NavigableDialogContextType<string> | undefined>' is not assignable to 'Context<NavigableDialogContextType<any>>'
const NavigableDialogContext: React.Context<NavigableDialogContextType<any>> =
  createNavigableDialogContext();

/**
 * Provides access to the current navigable dialog context.
 *
 * @returns The active step ID and a function to update it.
 *
 * @throws {Error} If called outside of a {@link NavigableDialogRoot} provider.
 */
export function useNavigableDialog<TStepName extends string>() {
  const context = useContext(NavigableDialogContext) as NavigableDialogContextType<TStepName>;
  if (context === undefined) {
    throw new Error("useNavigableDialog must be used within a NavigableDialogProvider");
  }
  return context;
}

// Helper type to extract valid step names when using the component
export type StepNamesFrom<T extends readonly { id: string }[]> = T[number]["id"];

/**
 * Provides context and structure for a multi-step navigable dialog.
 *
 * Wraps children with dialog UI and manages the active step state, exposing it via context to descendant components.
 *
 * @param children - Dialog content, typically including navigation and step content components.
 * @param isOpen - Controls whether the dialog is open.
 * @param onOpenChange - Callback invoked when the dialog open state changes.
 * @param dialogClassName - Optional additional class names for the dialog container.
 * @param preventAutoFocus - If true, prevents autofocus when the dialog opens. Defaults to true.
 */
export function NavigableDialogRoot<TStepName extends string>({
  children,
  isOpen,
  onOpenChange,
  dialogClassName,
  preventAutoFocus = true,
}: {
  children: ReactNode;
  isOpen: boolean;
  onOpenChange: (value: boolean) => void;
  dialogClassName?: string;
  preventAutoFocus?: boolean;
}) {
  // Internal state - we'll initialize this when we get the first items from Nav
  const [activeId, setActiveId] = useState<TStepName | undefined>();

  const contextValue = {
    activeId,
    setActiveId,
  };

  return (
    <NavigableDialogContext.Provider value={contextValue}>
      <Dialog open={isOpen} onOpenChange={onOpenChange}>
        <DialogContent
          className={cn(
            "drop-shadow-2xl border-grayA-4 overflow-hidden !rounded-2xl p-0 gap-0",
            dialogClassName,
          )}
          onOpenAutoFocus={(e) => {
            if (preventAutoFocus) {
              e.preventDefault();
            }
          }}
        >
          {children}
        </DialogContent>
      </Dialog>
    </NavigableDialogContext.Provider>
  );
}

/**
 * Renders the header section of a navigable dialog with a title and optional subtitle.
 *
 * @param title - The main title displayed in the dialog header.
 * @param subTitle - An optional subtitle displayed below the title.
 */
export function NavigableDialogHeader({
  title,
  subTitle,
}: {
  title: string;
  subTitle?: string;
}) {
  return <DefaultDialogHeader title={title} subTitle={subTitle} />;
}

/**
 * Renders the footer section of a navigable dialog.
 *
 * Displays arbitrary child elements within a styled dialog footer container.
 */
export function NavigableDialogFooter({ children }: { children: ReactNode }) {
  return <DefaultDialogFooter>{children}</DefaultDialogFooter>;
}

/**
 * Renders a vertical navigation sidebar for a multi-step dialog, allowing users to switch between steps.
 *
 * Each navigation item can display a label and optional icon, and can be disabled to prevent interaction. Navigation can be conditionally validated via an optional callback before switching steps.
 *
 * @param items - List of navigation items, each with an {@link id}, label, and optional icon.
 * @param onNavigate - Optional callback invoked before navigating away from the current step; must return `true` to allow navigation.
 * @param initialSelectedId - Optional ID of the step to select initially.
 * @param disabledIds - Optional list of step IDs to disable in the navigation.
 * @param navWidthClass - Optional CSS class for the navigation sidebar width.
 * @param className - Optional additional CSS classes for the sidebar container.
 */
export function NavigableDialogNav<TStepName extends string>({
  items,
  className,
  onNavigate,
  initialSelectedId,
  disabledIds,
  navWidthClass = "w-[220px]",
}: {
  items: {
    id: TStepName;
    label: ReactNode;
    icon?: FC<IconProps>;
  }[];
  className?: string;
  onNavigate?: (fromId: TStepName) => boolean | Promise<boolean>;
  initialSelectedId?: TStepName;
  disabledIds?: TStepName[];
  navWidthClass?: string;
}) {
  const { activeId, setActiveId } = useNavigableDialog<TStepName>();

  // Initialize activeId if it's not set and we have items
  useEffect(() => {
    if (activeId === undefined && items.length > 0) {
      setActiveId(initialSelectedId ?? items[0].id);
    }
  }, [activeId, items, initialSelectedId, setActiveId]);

  const handleItemNavigation = useCallback(
    async (newId: TStepName) => {
      // Skip if navigating to the same tab
      if (newId === activeId) {
        return;
      }

      // If onNavigate is provided, use it to validate navigation
      if (onNavigate && activeId) {
        const canNavigate = await onNavigate(activeId);
        if (canNavigate) {
          setActiveId(newId);
        }
      } else {
        // No validation needed, just navigate
        setActiveId(newId);
      }
    },
    [activeId, onNavigate, setActiveId],
  );

  return (
    <div
      className={cn(
        "border-r border-grayA-4 bg-white dark:bg-black p-6 flex flex-col items-start justify-start gap-3",
        "flex-shrink-0",
        navWidthClass,
        className,
      )}
    >
      {items.map((item) => {
        // Check if item is in disabled list
        const isDisabled = disabledIds?.includes(item.id);

        const IconComponent = item.icon;
        const isActive = item.id === activeId;

        return (
          <Button
            key={item.id}
            variant="outline"
            className={cn(
              "rounded-lg w-full px-3 py-1 [&>*:first-child]:justify-start focus:ring-0 [&_svg]:size-auto hover:bg-grayA-3 border-none",
              isActive ? "bg-grayA-3" : "",
              isDisabled && "opacity-50 cursor-not-allowed pointer-events-none",
            )}
            size="md"
            onClick={() => !isDisabled && handleItemNavigation(item.id)}
            disabled={isDisabled}
            aria-disabled={isDisabled}
          >
            {IconComponent && (
              <div>
                <IconComponent
                  size="md-regular"
                  className={cn(
                    isDisabled ? "text-gray-7" : isActive ? "text-gray-12" : "text-gray-9",
                  )}
                />
              </div>
            )}
            <span
              className={cn(
                "font-medium text-[13px] leading-[24px] w-full text-start",
                isDisabled ? "text-gray-7" : "text-gray-12",
              )}
            >
              {item.label}
            </span>
          </Button>
        );
      })}
    </div>
  );
}

/**
 * Renders the content area of a navigable dialog, displaying only the content for the currently active step.
 *
 * @param items - Array of step objects, each with an {@link id} and corresponding {@link content}.
 * @param className - Optional CSS class for custom styling.
 */
export function NavigableDialogContent<TStepName extends string>({
  items,
  className,
}: {
  items: {
    id: TStepName;
    content: ReactNode;
  }[];
  className?: string;
}) {
  const { activeId } = useNavigableDialog<TStepName>();

  return (
    <div className="flex-1 min-w-0 overflow-y-auto">
      <DefaultDialogContentArea className={cn(className)}>
        {items.map((item) => (
          <div key={item.id} className={cn("w-full", item.id !== activeId && "hidden")}>
            {item.content}
          </div>
        ))}
      </DefaultDialogContentArea>
    </div>
  );
}

/**
 * Provides a flex container for arranging the dialog's navigation and content areas.
 *
 * Renders its children inside a horizontally oriented, overflow-hidden layout.
 */
export function NavigableDialogBody({ children }: { children: ReactNode }) {
  return <div className="flex overflow-hidden">{children}</div>;
}
