"use client";
// biome-ignore lint: React in this context is used throughout, so biome will change to types because no APIs are used even though React is needed.
import * as React from "react";
import type { PropsWithChildren, ReactNode } from "react";
import { cn } from "../../lib/utils";
import { Dialog, DialogContent } from "./dialog";
import {
  DefaultDialogContentArea,
  DefaultDialogFooter,
  DefaultDialogHeader,
} from "./parts/dialog-parts";

type DialogContainerProps = PropsWithChildren<{
  className?: string;
  isOpen: boolean;
  onOpenChange: (value: boolean) => void;
  title: string;
  footer?: ReactNode;
  contentClassName?: string;
  preventAutoFocus?: boolean;
  subTitle?: string;
  showCloseWarning?: boolean;
  onAttemptClose?: () => void;
  modal?: boolean;
  preventOutsideClose?: boolean;
}>;

const DialogContainer = ({
  className,
  isOpen,
  subTitle,
  onOpenChange,
  title,
  children,
  footer,
  contentClassName,
  preventAutoFocus = false,
  showCloseWarning = false,
  onAttemptClose,
  modal = true,
  preventOutsideClose = false,
}: DialogContainerProps) => {
  return (
    <Dialog open={isOpen} onOpenChange={onOpenChange} modal={modal}>
      <DialogContent
        className={cn(
          "drop-shadow-2xl transform-gpu border-gray-4 max-h-[90vh] overflow-hidden rounded-2xl! p-0 gap-0 flex flex-col grow",
          "w-[90%] md:w-[70%] lg:w-[70%] xl:w-[50%] 2xl:w-[45%] max-w-[600px] max-h-[90vh] sm:max-h-[90vh] md:max-h-[70vh] lg:max-h-[90vh] xl:max-h-[80vh]",
          className,
        )}
        onOpenAutoFocus={(e) => {
          if (preventAutoFocus) {
            e.preventDefault();
          }
        }}
        showCloseWarning={showCloseWarning}
        onAttemptClose={onAttemptClose}
        preventOutsideClose={preventOutsideClose}
      >
        <DefaultDialogHeader title={title} subTitle={subTitle} />
        <DefaultDialogContentArea className={contentClassName}>{children}</DefaultDialogContentArea>
        {footer && <DefaultDialogFooter>{footer}</DefaultDialogFooter>}
      </DialogContent>
    </Dialog>
  );
};
DialogContainer.displayName = "DialogContainer";

export { DialogContainer };
