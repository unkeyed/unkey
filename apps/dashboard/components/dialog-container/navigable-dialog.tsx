"use client";
import { Dialog, DialogContent } from "@/components/ui/dialog";
import type { IconProps } from "@unkey/icons/src/props";
import { Button } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import { useCallback, useState } from "react";
import type { FC, ReactNode } from "react";
import { DefaultDialogContentArea, DefaultDialogFooter, DefaultDialogHeader } from "./dialog-parts";

export type NavItem<TStepName extends string> = {
  id: TStepName;
  label: ReactNode;
  icon?: FC<IconProps>;
  content: ReactNode;
};

type NavigableDialogProps<TStepName extends string> = {
  isOpen: boolean;
  onOpenChange: (value: boolean) => void;
  title: string;
  subTitle?: string;
  footer?: ReactNode;
  items: NavItem<TStepName>[];
  initialSelectedId?: TStepName;
  dialogClassName?: string;
  navClassName?: string;
  contentClassName?: string;
  preventAutoFocus?: boolean;
  navWidthClass?: string;
  onNavigate?: (fromId: TStepName) => boolean | Promise<boolean>;
};

export const NavigableDialog = <TStepName extends string>({
  isOpen,
  onOpenChange,
  title,
  subTitle,
  footer,
  items,
  initialSelectedId,
  dialogClassName,
  navClassName,
  contentClassName,
  preventAutoFocus = true,
  navWidthClass = "w-[220px]",
  onNavigate,
}: NavigableDialogProps<TStepName>) => {
  const [activeId, setActiveId] = useState<TStepName | undefined>(
    initialSelectedId ?? items[0]?.id,
  );

  const handleNavigation = useCallback(
    async (newId: TStepName) => {
      // Skip validation if navigating to the same tab
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
    [activeId, onNavigate],
  );

  return (
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
        <DefaultDialogHeader title={title} subTitle={subTitle} />
        <div className="flex overflow-hidden">
          <div
            className={cn(
              "border-r border-grayA-4 bg-white dark:bg-black p-6 flex flex-col items-start justify-start gap-3",
              "flex-shrink-0",
              navWidthClass,
              navClassName,
            )}
          >
            {items.map((item) => {
              const IconComponent = item.icon;
              const isActive = item.id === activeId;
              return (
                <Button
                  key={item.id}
                  variant={isActive ? "outline" : "ghost"}
                  className={cn(
                    "rounded-lg w-full px-3 py-1 [&>*:first-child]:justify-start focus:ring-0 [&_svg]:size-auto",
                    isActive ? "bg-grayA-2 focus:border-grayA-6" : "border border-transparent",
                  )}
                  size="md"
                  onClick={() => handleNavigation(item.id)}
                >
                  {IconComponent && (
                    <div>
                      <IconComponent
                        size="md-regular"
                        className={cn(isActive ? "text-gray-12" : "text-gray-9")}
                      />
                    </div>
                  )}
                  <span
                    className={cn(
                      "font-medium text-[13px] leading-[24px] text-gray-12 w-full text-start",
                    )}
                  >
                    {item.label}
                  </span>
                </Button>
              );
            })}
          </div>
          {/* Right Content Pane Wrapper */}
          <div className="flex-1 min-w-0 overflow-y-auto">
            <DefaultDialogContentArea className={cn(contentClassName)}>
              {items.map((item) => (
                <div key={item.id} className={cn("w-full", item.id !== activeId && "hidden")}>
                  {item.content}
                </div>
              ))}
            </DefaultDialogContentArea>
          </div>
        </div>
        {footer && <DefaultDialogFooter>{footer}</DefaultDialogFooter>}
      </DialogContent>
    </Dialog>
  );
};
