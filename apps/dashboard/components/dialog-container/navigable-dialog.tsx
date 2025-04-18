"use client";

import { Dialog, DialogContent } from "@/components/ui/dialog";
import type { IconProps } from "@unkey/icons/src/props";
import { Button } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import { useState } from "react";
import type { FC, ReactNode } from "react";
import { DefaultDialogContentArea, DefaultDialogFooter, DefaultDialogHeader } from "./dialog-parts";

export type NavItem = {
  id: string;
  label: string;
  icon?: FC<IconProps>;
  content: ReactNode;
};

type NavigableDialogProps = {
  isOpen: boolean;
  onOpenChange: (value: boolean) => void;
  title: string;
  subTitle?: string;
  footer?: ReactNode;
  items: NavItem[];
  initialSelectedId?: string;
  dialogClassName?: string;
  navClassName?: string;
  contentClassName?: string;
  preventAutoFocus?: boolean;
  navWidthClass?: string;
};

export const NavigableDialog = ({
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
}: NavigableDialogProps) => {
  const [activeId, setActiveId] = useState<string | undefined>(initialSelectedId ?? items[0]?.id);

  // No longer finding just the activeItem here for rendering content

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
                    "rounded-lg w-full px-3 py-1 [&>*:first-child]:justify-start focus:ring-0",
                    isActive ? "bg-grayA-2 focus:border-grayA-6" : "border border-transparent",
                  )}
                  size="md"
                  onClick={() => setActiveId(item.id)} // Only updates the activeId state
                >
                  {IconComponent && (
                    <IconComponent
                      size="sm-regular"
                      className={cn(isActive ? "text-gray-12" : "text-gray-9")}
                    />
                  )}
                  <span className={cn("font-medium text-[13px] leading-[24px] text-gray-12")}>
                    {item.label}
                  </span>
                </Button>
              );
            })}
          </div>

          {/* Right Content Pane Wrapper */}
          <div className="flex-1 min-w-0 overflow-y-auto">
            {" "}
            <DefaultDialogContentArea className={cn(contentClassName)}>
              {items.map((item) => (
                <div
                  key={item.id}
                  className={cn(
                    "w-full",
                    item.id !== activeId && "hidden", // "hidden" applies `display: none`
                  )}
                >
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
