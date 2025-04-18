"use client";
import { Dialog, DialogContent } from "@/components/ui/dialog";
import { cn } from "@unkey/ui/src/lib/utils";
import type { PropsWithChildren, ReactNode } from "react";
import {
  DefaultDialogHeader,
  DefaultDialogContentArea,
  DefaultDialogFooter,
} from "./dialog-parts";

type DialogContainerProps = PropsWithChildren<{
  className?: string;
  isOpen: boolean;
  onOpenChange: (value: boolean) => void;
  title: string;
  footer?: ReactNode;
  contentClassName?: string;
  preventAutoFocus?: boolean;
  subTitle?: string;
}>;

export const DialogContainer = ({
  className,
  isOpen,
  subTitle,
  onOpenChange,
  title,
  children,
  footer,
  contentClassName,
  preventAutoFocus = true,
}: DialogContainerProps) => {
  return (
    <Dialog open={isOpen} onOpenChange={onOpenChange}>
      <DialogContent
        className={cn(
          "drop-shadow-2xl border-gray-4 overflow-hidden !rounded-2xl p-0 gap-0",
          className
        )}
        onOpenAutoFocus={(e) => {
          if (preventAutoFocus) {
            e.preventDefault();
          }
        }}
      >
        <DefaultDialogHeader title={title} subTitle={subTitle} />

        <DefaultDialogContentArea className={contentClassName}>
          {children}
        </DefaultDialogContentArea>

        {footer && <DefaultDialogFooter>{footer}</DefaultDialogFooter>}
      </DialogContent>
    </Dialog>
  );
};

export { DefaultDialogHeader, DefaultDialogContentArea, DefaultDialogFooter };
