"use client";

import { Dialog as DialogPrimitive } from "@base-ui/react/dialog";
import { IconXmarkOutline18 } from "@unkey/icons";
import * as React from "react";
import { cn } from "../lib/utils";
import { DiscardChangesDialog } from "./dialog/discard-changes-dialog";
import { UnsavedChangesScope, useReportUnsavedChanges, useUnsavedChanges } from "./unsaved-changes";

type SlidePanelBackdrop = "blur" | "dim" | "none";

export type SlidePanelProps = {
  children: React.ReactNode;
  isOpen: boolean;
  onClose: () => void;
  /** Closing asks before the edits are lost. Forms inside the panel report their own dirty state. */
  dirty?: boolean;
  onExitComplete?: () => void;
  side?: "left" | "right";
  widthClassName?: string;
  className?: string;
  backdrop?: SlidePanelBackdrop;
  fitContent?: boolean;
  "data-docs-target"?: string;
};

export function SlidePanel({
  children,
  isOpen,
  onClose,
  dirty = false,
  onExitComplete,
  side = "right",
  widthClassName = "w-175",
  className,
  backdrop = "blur",
  fitContent = false,
  "data-docs-target": docsTarget,
}: SlidePanelProps) {
  const [confirming, setConfirming] = React.useState(false);
  const inner = useUnsavedChanges();
  const holdsChanges = isOpen && (dirty || inner.isDirty);
  useReportUnsavedChanges(holdsChanges);
  if (!isOpen && confirming) {
    setConfirming(false);
  }
  const requestClose = () => {
    if (holdsChanges) {
      setConfirming(true);
    } else {
      onClose();
    }
  };

  return (
    <DialogPrimitive.Root
      open={isOpen}
      modal={false}
      disablePointerDismissal
      onOpenChange={(open, eventDetails) => {
        if (open) {
          return;
        }
        // Opening a dialog from inside the panel moves focus out of it, which Base UI reports
        // as a dismissal. The panel should outlive anything it opens.
        if (eventDetails.reason === "focus-out") {
          eventDetails.cancel();
          return;
        }
        requestClose();
      }}
      onOpenChangeComplete={(open) => {
        if (!open) {
          onExitComplete?.();
        }
      }}
    >
      <DialogPrimitive.Portal>
        {backdrop !== "none" && (
          <DialogPrimitive.Backdrop
            onClick={requestClose}
            className={cn(
              "fixed inset-0 z-50 transition-opacity duration-300 ease-[cubic-bezier(0.4,0,0.2,1)] motion-reduce:transition-none",
              "data-starting-style:opacity-0 data-ending-style:opacity-0",
              backdrop === "blur" ? "bg-background/5 backdrop-blur-[2px]" : "bg-background/20",
            )}
          />
        )}
        <DialogPrimitive.Popup
          data-slide-panel-open=""
          data-docs-target={docsTarget}
          className={cn(
            "[--slide-panel-inset:0.75rem]",
            "fixed z-50 flex flex-col shadow-floating",
            "rounded-xl bg-raised",
            "top-(--slide-panel-inset) bottom-(--slide-panel-inset)",
            "max-w-[calc(100dvw_-_var(--slide-panel-inset)_*_2)]",
            side === "right" ? "right-(--slide-panel-inset)" : "left-(--slide-panel-inset)",
            fitContent && "bottom-auto max-h-[calc(100dvh_-_var(--slide-panel-inset)_*_2)]",
            "transition-[opacity,translate] duration-200 ease-[cubic-bezier(0.4,0,0.2,1)] motion-reduce:transition-none",
            "data-starting-style:opacity-0 data-ending-style:opacity-0",
            side === "right"
              ? "data-starting-style:translate-x-5 data-ending-style:translate-x-10"
              : "data-starting-style:-translate-x-5 data-ending-style:-translate-x-10",
            widthClassName,
            className,
          )}
        >
          <div className="flex min-h-0 flex-1 flex-col overflow-hidden rounded-xl">
            <UnsavedChangesScope report={inner.report}>{children}</UnsavedChangesScope>
          </div>
          <DiscardChangesDialog
            open={confirming}
            onKeepEditing={() => setConfirming(false)}
            onDiscard={() => {
              setConfirming(false);
              onClose();
            }}
          />
        </DialogPrimitive.Popup>
      </DialogPrimitive.Portal>
    </DialogPrimitive.Root>
  );
}

export type SlidePanelHeaderProps = {
  children: React.ReactNode;
  className?: string;
};

export function SlidePanelHeader({ children, className }: SlidePanelHeaderProps) {
  return (
    <div className={cn("flex items-start justify-between px-6 pt-6 pb-2", className)}>
      {children}
    </div>
  );
}

export type SlidePanelTitleProps = DialogPrimitive.Title.Props & {
  ref?: React.Ref<HTMLHeadingElement>;
};

export function SlidePanelTitle({ className, ...props }: SlidePanelTitleProps) {
  return (
    <DialogPrimitive.Title
      className={cn("text-lg font-semibold leading-tight tracking-tight text-gray-12", className)}
      {...props}
    />
  );
}

export type SlidePanelDescriptionProps = DialogPrimitive.Description.Props & {
  ref?: React.Ref<HTMLParagraphElement>;
};

export function SlidePanelDescription({ className, ...props }: SlidePanelDescriptionProps) {
  return (
    <DialogPrimitive.Description
      className={cn("text-sm leading-5 text-gray-11", className)}
      {...props}
    />
  );
}

export type SlidePanelContentProps = {
  children: React.ReactNode;
  className?: string;
};

export function SlidePanelContent({ children, className }: SlidePanelContentProps) {
  return <div className={cn("min-h-0 flex-1", className)}>{children}</div>;
}

export type SlidePanelFooterProps = {
  children: React.ReactNode;
  className?: string;
};

export function SlidePanelFooter({ children, className }: SlidePanelFooterProps) {
  return <div className={cn("border-t px-6 py-4", className)}>{children}</div>;
}

export const SlidePanelClose = DialogPrimitive.Close;

export type SlidePanelCloseButtonProps = DialogPrimitive.Close.Props & {
  ref?: React.Ref<HTMLButtonElement>;
};

export function SlidePanelCloseButton({ className, ...props }: SlidePanelCloseButtonProps) {
  return (
    <DialogPrimitive.Close
      aria-label="Close panel"
      className={cn(
        "inline-flex size-9 shrink-0 cursor-pointer items-center justify-center rounded-md text-gray-10 transition-colors hover:bg-grayA-3 hover:text-gray-12",
        className,
      )}
      {...props}
    >
      <IconXmarkOutline18 className="size-4" />
    </DialogPrimitive.Close>
  );
}

export type { SlidePanelBackdrop };
