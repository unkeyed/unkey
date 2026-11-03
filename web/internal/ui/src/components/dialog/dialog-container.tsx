"use client";
// biome-ignore lint/correctness/noUnusedImports: the package compiles JSX with the classic runtime ("jsx": "react"), so React must be in scope.
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
  "data-docs-target"?: string;
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
  "data-docs-target": docsTarget,
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
        data-docs-target={docsTarget}
        className={cn(
          "max-h-[90vh] overflow-hidden p-0 gap-0 flex flex-col grow",
          "w-[90%] max-w-[480px] md:max-h-[70vh] lg:max-h-[90vh] xl:max-h-[80vh]",
          className,
        )}
        initialFocus={preventAutoFocus ? false : undefined}
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
