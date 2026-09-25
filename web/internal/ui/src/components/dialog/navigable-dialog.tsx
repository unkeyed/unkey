"use client";
import type { IconProps } from "@unkey/icons";
// biome-ignore lint/style/useImportType: the package compiles JSX with the classic runtime, so React must be in scope as a value.
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

// @ts-expect-error Type 'Context<NavigableDialogContextType<string> | undefined>' is not assignable to 'Context<NavigableDialogContextType<any>>'
// biome-ignore lint/suspicious/noExplicitAny: safe to leave
const NavigableDialogContext: React.Context<NavigableDialogContextType<any>> =
  createNavigableDialogContext();

const useNavigableDialog = <TStepName extends string>() => {
  const context = useContext(NavigableDialogContext) as NavigableDialogContextType<TStepName>;
  if (context === undefined) {
    throw new Error("useNavigableDialog must be used within a NavigableDialogProvider");
  }
  return context;
};

export type StepNamesFrom<T extends readonly { id: string }[]> = T[number]["id"];

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
              "overflow-hidden rounded-2xl! p-0 gap-0 flex flex-col max-h-[90vh]",
              dialogClassName,
            )}
            initialFocus={preventAutoFocus ? false : undefined}
          >
            {children}
          </DialogContent>
        </DialogPortal>
      </Dialog>
    </NavigableDialogContext.Provider>
  );
};

const NavigableDialogHeader = ({
  title,
  subTitle,
}: {
  title: string;
  subTitle?: string;
}) => {
  return <DefaultDialogHeader title={title} subTitle={subTitle} />;
};

const NavigableDialogFooter = ({ children }: { children: ReactNode }) => {
  return <DefaultDialogFooter>{children}</DefaultDialogFooter>;
};

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
      if (newId === activeId) {
        return;
      }

      if (onNavigate && activeId) {
        const canNavigate = await onNavigate(activeId);
        if (canNavigate) {
          setActiveId(newId);
        }
      } else {
        setActiveId(newId);
      }
    },
    [activeId, onNavigate, setActiveId],
  );

  return (
    <div
      className={cn(
        "border-r bg-raised p-6 flex flex-col items-start justify-start gap-3",
        "shrink-0",
        navWidthClass,
        className,
      )}
    >
      {items.map((item) => {
        const isDisabled = disabledIds?.includes(item.id);

        const IconComponent = item.icon;
        const isActive = item.id === activeId;

        return (
          <Button
            key={item.id}
            variant="outline"
            className={cn(
              "rounded-lg w-full px-3 py-1 [&>*:first-child]:justify-start focus:ring-0 [&_svg]:size-3.5 hover:bg-grayA-3 border-none",
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
                  className={cn(
                    "size-3.5",
                    isDisabled ? "text-gray-7" : isActive ? "text-gray-12" : "text-gray-9",
                  )}
                />
              </div>
            )}
            <span
              className={cn(
                "font-medium text-sm leading-[24px] w-full text-start",
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

const NavigableDialogBody = ({
  children,
  className,
}: {
  children: ReactNode;
  className?: string;
}) => {
  return (
    <div className={cn("flex grow overflow-x-hidden overflow-y-hidden", className)}>{children}</div>
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
