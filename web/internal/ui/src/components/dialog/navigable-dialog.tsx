"use client";
import type { IconProps } from "@unkey/icons/src/props";
// biome-ignore lint: React in this context is used throughout, so biome will change to types because no APIs are used even though React is needed.
import * as React from "react";
import { createContext, useCallback, useContext, useEffect, useState } from "react";
import type { FC, ReactNode } from "react";
import { cn } from "../../lib/utils";
import { Button } from "../buttons/button";
import { Dialog, DialogContent, DialogPortal } from "./dialog";
import {
  DefaultDialogContentArea,
  DefaultDialogFooter,
  DefaultDialogHeader,
} from "./parts/dialog-parts";

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
// biome-ignore lint/suspicious/noExplicitAny: safe to leave
const NavigableDialogContext: React.Context<NavigableDialogContextType<any>> =
  createNavigableDialogContext();

// Hook to use the NavigableDialog context
const useNavigableDialog = <TStepName extends string>() => {
  const context = useContext(NavigableDialogContext) as NavigableDialogContextType<TStepName>;
  if (context === undefined) {
    throw new Error("useNavigableDialog must be used within a NavigableDialogProvider");
  }
  return context;
};

// Helper type to extract valid step names when using the component
export type StepNamesFrom<T extends readonly { id: string }[]> = T[number]["id"];

// Root component that provides context and structure
const NavigableDialogRoot = <TStepName extends string>({
  children,
  isOpen,
  onOpenChange,
  dialogClassName,
  preventAutoFocus = false,
}: {
  children: ReactNode;
  isOpen: boolean;
  onOpenChange: (value: boolean) => void;
  dialogClassName?: string;
  preventAutoFocus?: boolean;
}) => {
  const [activeId, setActiveId] = useState<TStepName | undefined>();

  const contextValue = {
    activeId,
    setActiveId,
  };

  return (
    <NavigableDialogContext.Provider value={contextValue}>
      <Dialog open={isOpen} onOpenChange={onOpenChange} modal={true}>
        <DialogPortal>
          <DialogContent
            onKeyDown={(e) => {
              // Allow keyboard events to propagate to nested components like Combobox
              if (
                e.key === "ArrowDown" ||
                e.key === "ArrowUp" ||
                e.key === "Enter" ||
                e.key === "Escape"
              ) {
                return;
              }
              e.stopPropagation();
            }}
            className={cn(
              "drop-shadow-2xl transform-gpu border-grayA-4 overflow-hidden !rounded-2xl p-0 gap-0 flex flex-col max-h-[90vh]",
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
        </DialogPortal>
      </Dialog>
    </NavigableDialogContext.Provider>
  );
};

// Header component
const NavigableDialogHeader = ({
  title,
  subTitle,
}: {
  title: string;
  subTitle?: string;
}) => {
  return <DefaultDialogHeader title={title} subTitle={subTitle} />;
};

// Footer component
const NavigableDialogFooter = ({ children }: { children: ReactNode }) => {
  return <DefaultDialogFooter>{children}</DefaultDialogFooter>;
};

// Navigation sidebar component
const NavigableDialogNav = <TStepName extends string>({
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
}) => {
  const { activeId, setActiveId } = useNavigableDialog<TStepName>();

  // Initialize activeId if it's not set and we have items
  useEffect(() => {
    const allIds = items.map((i) => i.id);
    if (!activeId || !allIds.includes(activeId)) {
      setActiveId(
        initialSelectedId && allIds.includes(initialSelectedId) ? initialSelectedId : allIds[0],
      );
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
                  iconSize="md-medium"
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
};

const NavigableDialogContent = <TStepName extends string>({
  items,
  className,
}: {
  items: {
    id: TStepName;
    content: ReactNode;
  }[];
  className?: string;
}) => {
  const { activeId } = useNavigableDialog<TStepName>();
  return (
    <div className="flex-1 min-w-0 overflow-y-auto">
      <DefaultDialogContentArea className={cn("min-h-[70vh] xl:min-h-[50vh] h-full", className)}>
        <div className="h-full relative overflow-visible">
          {items.map((item) => {
            const isActive = item.id === activeId;
            return (
              <div
                key={item.id}
                className={cn(
                  "w-full absolute inset-0 overflow-y-auto scrollbar-hide",
                  "transition-all duration-300 ease-out",
                  isActive
                    ? "opacity-100 translate-x-0 z-10"
                    : "opacity-0 translate-x-5 z-0 pointer-events-none",
                )}
                aria-hidden={!isActive}
              >
                <div className="h-full">{item.content}</div>
              </div>
            );
          })}
        </div>
      </DefaultDialogContentArea>
    </div>
  );
};

// Main container for the nav and content
const NavigableDialogBody = ({
  children,
  className,
}: {
  children: ReactNode;
  className?: string;
}) => {
  return (
    <div className={cn("flex flex-grow overflow-x-hidden overflow-y-hidden", className)}>
      {children}
    </div>
  );
};

NavigableDialogBody.displayName = "NavigableDialogBody";
NavigableDialogContent.displayName = "NavigableDialogContent";
NavigableDialogNav.displayName = "NavigableDialogNav";
NavigableDialogHeader.displayName = "NavigableDialogHeader";
NavigableDialogFooter.displayName = "NavigableDialogFooter";
NavigableDialogRoot.displayName = "NavigableDialogRoot";

export {
  NavigableDialogRoot,
  NavigableDialogHeader,
  NavigableDialogFooter,
  NavigableDialogNav,
  NavigableDialogContent,
  NavigableDialogBody,
  useNavigableDialog,
};
