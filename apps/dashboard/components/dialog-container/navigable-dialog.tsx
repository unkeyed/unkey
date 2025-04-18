"use client";

import { useState } from "react";
import { Dialog, DialogContent } from "@/components/ui/dialog";
import { Button } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import type { ReactNode } from "react";
import {
  DefaultDialogContentArea,
  DefaultDialogFooter,
  DefaultDialogHeader,
} from "./dialog-parts";

// Define the structure for a navigation item (same as before)
export type NavItem = {
  id: string;
  label: string;
  icon?: React.ElementType<{ size?: string | number; className?: string }>;
  content: ReactNode; // The content component/JSX for this item
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
  contentClassName?: string; // Applied to DefaultDialogContentArea
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
  const [activeId, setActiveId] = useState<string | undefined>(
    initialSelectedId ?? items[0]?.id
  );

  // No longer finding just the activeItem here for rendering content

  return (
    <Dialog open={isOpen} onOpenChange={onOpenChange}>
      <DialogContent
        className={cn(
          "drop-shadow-2xl border-grayA-4 overflow-hidden !rounded-2xl p-0 gap-0",
          dialogClassName
        )}
        onOpenAutoFocus={(e) => {
          if (preventAutoFocus) {
            e.preventDefault();
          }
        }}
      >
        <DefaultDialogHeader title={title} subTitle={subTitle} />

        <div className="flex overflow-hidden">
          {/* Left Navigation Pane (Same as before) */}
          <div
            className={cn(
              "border-r border-grayA-4 bg-white dark:bg-black p-6 flex flex-col items-start justify-start gap-3",
              "flex-shrink-0",
              navWidthClass,
              navClassName
            )}
          >
            {items.map((item) => {
              const IconComponent = item.icon;
              const isActive = item.id === activeId;
              return (
                <Button
                  key={item.id}
                  variant={isActive ? "primary" : "ghost"}
                  className={cn(
                    "rounded-lg w-full px-3 py-1 [&>*:first-child]:justify-start",
                    isActive ? "bg-accent-2 dark:bg-grayA-3" : ""
                  )}
                  size="md"
                  onClick={() => setActiveId(item.id)} // Only updates the activeId state
                >
                  {IconComponent && (
                    <IconComponent size={16} className="mr-2 text-gray-11" />
                  )}
                  <span className="font-medium text-[13px] leading-[24px] text-gray-12">
                    {item.label}
                  </span>
                </Button>
              );
            })}
          </div>

          {/* Right Content Pane Wrapper */}
          <div className="flex-1 min-w-0 overflow-y-auto">
            {" "}
            {/* Added overflow-y-auto */}
            {/* DefaultDialogContentArea wraps the whole switchable area */}
            <DefaultDialogContentArea className={cn(contentClassName)}>
              {/* Render ALL item contents, hide inactive ones with CSS */}
              {items.map((item) => (
                <div
                  key={item.id}
                  className={cn(
                    // Basic styling for the container if needed
                    "w-full",
                    // Hide the div if its item is not the active one
                    item.id !== activeId && "hidden" // "hidden" applies `display: none`
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

// Export the parts if needed from here too
export { DefaultDialogContentArea, DefaultDialogFooter, DefaultDialogHeader };
