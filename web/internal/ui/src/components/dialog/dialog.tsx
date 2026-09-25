"use client";

import { Dialog as DialogPrimitive } from "@base-ui/react/dialog";
import { IconXmarkOutline18 } from "@unkey/icons";
import * as React from "react";

import { cn } from "../../lib/utils";
import { Button } from "../buttons/button";

// Base UI surfaces dismissal reasons (escape, outside press, focus out) only
// on `Root.onOpenChange`, so `DialogContent` publishes its dismiss props up
// through this ref.
type DialogDismissConfig = {
  showCloseWarning: boolean;
  onAttemptClose?: () => void;
  preventOutsideClose: boolean;
};

const DialogDismissContext =
  React.createContext<React.MutableRefObject<DialogDismissConfig> | null>(null);

type DialogProps = Omit<DialogPrimitive.Root.Props, "children"> & {
  children?: React.ReactNode;
};

const Dialog = ({ children, onOpenChange, ...props }: DialogProps) => {
  const dismissRef = React.useRef<DialogDismissConfig>({
    showCloseWarning: false,
    onAttemptClose: undefined,
    preventOutsideClose: false,
  });

  const handleOpenChange = React.useCallback(
    (open: boolean, eventDetails: DialogPrimitive.Root.ChangeEventDetails) => {
      if (!open) {
        const { showCloseWarning, onAttemptClose, preventOutsideClose } = dismissRef.current;
        const { reason } = eventDetails;
        const isOutside = reason === "outside-press" || reason === "focus-out";

        // Combobox listboxes and cmdk roots portal outside the dialog subtree,
        // so Base UI reports them as outside presses.
        if (isOutside) {
          const target = eventDetails.event?.target as HTMLElement | null;
          if (
            target?.closest('[role="listbox"]') ||
            target?.closest("[data-combobox-popup]") ||
            target?.closest("[cmdk-root]")
          ) {
            eventDetails.cancel();
            return;
          }
        }

        if (preventOutsideClose && isOutside) {
          eventDetails.cancel();
          return;
        }

        if (showCloseWarning && (reason === "escape-key" || isOutside)) {
          eventDetails.cancel();
          onAttemptClose?.();
          return;
        }
      }
      onOpenChange?.(open, eventDetails);
    },
    [onOpenChange],
  );

  return (
    <DialogDismissContext.Provider value={dismissRef}>
      <DialogPrimitive.Root onOpenChange={handleOpenChange} {...props}>
        {children}
      </DialogPrimitive.Root>
    </DialogDismissContext.Provider>
  );
};
Dialog.displayName = "Dialog";

const DialogTrigger = DialogPrimitive.Trigger;

const DialogPortal = DialogPrimitive.Portal;

const DialogClose = DialogPrimitive.Close;

function DialogOverlay({
  className,
  showCloseWarning: _showCloseWarning,
  onAttemptClose: _onAttemptClose,
  ref,
  ...props
}: DialogPrimitive.Backdrop.Props & {
  showCloseWarning?: boolean;
  onAttemptClose?: () => void;
  ref?: React.Ref<React.ComponentRef<typeof DialogPrimitive.Backdrop>>;
}) {
  return (
    <DialogPrimitive.Backdrop
      ref={ref}
      className={cn(
        "fixed inset-0 z-50 bg-black/30 backdrop-blur-xs transition-opacity data-starting-style:opacity-0 data-ending-style:opacity-0",
        className,
      )}
      {...props}
    />
  );
}

function DialogContent({
  className,
  children,
  showCloseWarning = false,
  onAttemptClose,
  xButtonRef,
  preventOutsideClose = false,
  ref,
  ...props
}: DialogPrimitive.Popup.Props & {
  showCloseWarning?: boolean;
  onAttemptClose?: () => void;
  xButtonRef?: React.RefObject<HTMLButtonElement>;
  preventOutsideClose?: boolean;
  ref?: React.Ref<React.ComponentRef<typeof DialogPrimitive.Popup>>;
}) {
  const dismissRef = React.useContext(DialogDismissContext);
  if (dismissRef) {
    dismissRef.current = { showCloseWarning, onAttemptClose, preventOutsideClose };
  }

  const handleCloseAttempt = React.useCallback(() => {
    if (showCloseWarning) {
      onAttemptClose?.();
    }
  }, [showCloseWarning, onAttemptClose]);

  const buttonClassNames =
    "absolute right-4 top-4 rounded-xs opacity-70 ring-offset-background transition-opacity hover:opacity-100 focus:outline-hidden focus:ring-2 focus:ring-gray-6 focus:ring-offset-2 focus:rounded-lg disabled:pointer-events-none data-open:bg-grayA-3 text-gray-11 z-51 [&_svg]:size-[14px] hover:rounded-lg";

  return (
    <DialogPortal>
      <DialogOverlay />
      <DialogPrimitive.Popup
        ref={ref}
        className={cn(
          "fixed left-[50%] top-[50%] z-50 grid w-full max-w-lg translate-x-[-50%] translate-y-[-50%] gap-4 bg-raised p-6 shadow-floating duration-200 transition-[opacity,scale,translate] data-starting-style:opacity-0 data-starting-style:scale-95 data-ending-style:opacity-0 data-ending-style:scale-95 sm:rounded-lg",
          className,
        )}
        onKeyDown={(e) => {
          if (e.key === "ArrowDown" || e.key === "ArrowUp" || e.key === "Enter") {
            return;
          }
          if (e.key === "Tab") {
            e.stopPropagation();
          }
        }}
        {...props}
      >
        {children}

        {showCloseWarning ? (
          <button
            ref={xButtonRef}
            type="button"
            onClick={handleCloseAttempt}
            className={buttonClassNames}
            aria-label="Close dialog with confirmation"
          >
            <IconXmarkOutline18 className="size-3.5" />
          </button>
        ) : (
          <DialogPrimitive.Close
            render={
              <Button
                size="icon"
                variant="ghost"
                type="button"
                className={buttonClassNames}
                aria-label="Close dialog"
              >
                <IconXmarkOutline18 className="size-4" />
              </Button>
            }
          />
        )}
      </DialogPrimitive.Popup>
    </DialogPortal>
  );
}

const DialogHeader = ({ className, ...props }: React.HTMLAttributes<HTMLDivElement>) => (
  <div className={cn("flex flex-col space-y-1.5 text-center sm:text-left", className)} {...props} />
);
DialogHeader.displayName = "DialogHeader";

const DialogFooter = ({ className, ...props }: React.HTMLAttributes<HTMLDivElement>) => (
  <div
    className={cn("flex flex-col-reverse sm:flex-row sm:justify-end sm:space-x-2", className)}
    {...props}
  />
);
DialogFooter.displayName = "DialogFooter";

function DialogTitle({
  className,
  ref,
  ...props
}: DialogPrimitive.Title.Props & {
  ref?: React.Ref<React.ComponentRef<typeof DialogPrimitive.Title>>;
}) {
  return (
    <DialogPrimitive.Title
      ref={ref}
      className={cn("text-lg font-semibold leading-none tracking-tight", className)}
      {...props}
    />
  );
}

function DialogDescription({
  className,
  ref,
  ...props
}: DialogPrimitive.Description.Props & {
  ref?: React.Ref<React.ComponentRef<typeof DialogPrimitive.Description>>;
}) {
  return (
    <DialogPrimitive.Description
      ref={ref}
      className={cn("text-sm text-gray-11", className)}
      {...props}
    />
  );
}

export {
  Dialog,
  DialogPortal,
  DialogOverlay,
  DialogClose,
  DialogTrigger,
  DialogContent,
  DialogHeader,
  DialogFooter,
  DialogTitle,
  DialogDescription,
};

export type { DialogProps };
