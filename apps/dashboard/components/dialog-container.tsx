"use client";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { cn } from "@unkey/ui/src/lib/utils";
import type { PropsWithChildren, ReactNode } from "react";

type DialogContainerProps = PropsWithChildren<{
  isOpen: boolean;
  onOpenChange: (value: boolean) => void;
  title: string;
  footer?: ReactNode;
  contentClassName?: string;
  preventAutoFocus?: boolean;
}>;

export const DialogContainer = ({
  isOpen,
  onOpenChange,
  title,
  children,
  footer,
  contentClassName = "bg-accent-2 dark:bg-grayA-2",
  preventAutoFocus = true,
}: DialogContainerProps) => {
  return (
    <Dialog open={isOpen} onOpenChange={onOpenChange}>
      <DialogContent
        className="drop-shadow-2xl border-gray-4 overflow-hidden !rounded-2xl p-0 gap-0"
        onOpenAutoFocus={(e) => {
          if (preventAutoFocus) {
            e.preventDefault();
          }
        }}
      >
        <DialogHeader className="border-b border-gray-4 bg-white dark:bg-black ">
          <DialogTitle className="px-6 py-4 text-gray-12 font-medium text-base">
            {title}
          </DialogTitle>
        </DialogHeader>
        <div
          className={cn("bg-grayA-2 flex flex-col gap-4 py-4 px-6 text-gray-11", contentClassName)}
        >
          {children}
        </div>
        {footer && (
          <DialogFooter className="p-6 border-t border-gray-4 bg-white dark:bg-black text-gray-9 ">
            {footer}
          </DialogFooter>
        )}
      </DialogContent>
    </Dialog>
  );
};
