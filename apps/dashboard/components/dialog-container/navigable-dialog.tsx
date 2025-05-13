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

// Hook to use the NavigableDialog context
export function useNavigableDialog<TStepName extends string>() {
  const context = useContext(NavigableDialogContext) as NavigableDialogContextType<TStepName>;
  if (context === undefined) {
    throw new Error("useNavigableDialog must be used within a NavigableDialogProvider");
  }
  return context;
}

// Helper type to extract valid step names when using the component
export type StepNamesFrom<T extends readonly { id: string }[]> = T[number]["id"];

// Root component that provides context and structure
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
            "drop-shadow-2xl border-grayA-4 overflow-hidden !rounded-2xl p-0 gap-0 flex flex-col max-h-[90vh]",
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

// Header component
export function NavigableDialogHeader({
  title,
  subTitle,
}: {
  title: string;
  subTitle?: string;
}) {
  return <DefaultDialogHeader title={title} subTitle={subTitle} />;
}

// Footer component
export function NavigableDialogFooter({ children }: { children: ReactNode }) {
  return <DefaultDialogFooter>{children}</DefaultDialogFooter>;
}

// Navigation sidebar component
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

// Content area component
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
      <DefaultDialogContentArea className={cn("min-h-[70vh] lg:min-h-[50vh]", className)}>
        {items.map((item) => (
          <div key={item.id} className={cn("w-full", item.id !== activeId && "hidden")}>
            {item.content}
          </div>
        ))}
      </DefaultDialogContentArea>
    </div>
  );
}

// Main container for the nav and content
export function NavigableDialogBody({
  children,
  className,
}: {
  children: ReactNode;
  className?: string;
}) {
  return <div className={cn("flex flex-grow overflow-hidden", className)}>{children}</div>;
}
