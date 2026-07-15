"use client";

import { Dialog as DialogPrimitive } from "@base-ui/react/dialog";
import { XMark } from "@unkey/icons";
import * as React from "react";

import { cn } from "../../lib/utils";
import { Button } from "../buttons/button";

/**
 * Dismiss coordination between `DialogContent` (which owns the
 * `showCloseWarning` / `preventOutsideClose` props) and the root
 * `onOpenChange` handler, which is where Base UI surfaces dismissal reasons
 * (escape key / outside press / focus out). Radix handled these per-part on
 * `Content`; Base UI consolidates them onto `Root.onOpenChange`.
 */
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

        // Keep the dialog open when the interaction happens inside a nested
        // portal such as a Combobox listbox or a cmdk root (these render
        // outside the dialog DOM subtree, so Base UI treats them as outside).
        if (isOutside) {
          const target = eventDetails.event?.target as HTMLElement | null;
          if (target?.closest('[role="listbox"]') || target?.closest("[cmdk-root]")) {
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

const DialogOverlay = React.forwardRef<
  React.ComponentRef<typeof DialogPrimitive.Backdrop>,
  DialogPrimitive.Backdrop.Props & {
    showCloseWarning?: boolean;
    onAttemptClose?: () => void;
  }
>(
  (
    { className, showCloseWarning: _showCloseWarning, onAttemptClose: _onAttemptClose, ...props },
    ref,
  ) => (
    <DialogPrimitive.Backdrop
      ref={ref}
      className={cn(
        "fixed inset-0 z-50 bg-black/30 backdrop-blur-xs transition-opacity data-starting-style:opacity-0 data-ending-style:opacity-0",
        className,
      )}
      {...props}
    />
  ),
);
DialogOverlay.displayName = "DialogOverlay";

const DialogContent = React.forwardRef<
  React.ComponentRef<typeof DialogPrimitive.Popup>,
  DialogPrimitive.Popup.Props & {
    showCloseWarning?: boolean;
    onAttemptClose?: () => void;
    xButtonRef?: React.RefObject<HTMLButtonElement>;
    preventOutsideClose?: boolean;
  }
>(
  (
    {
      className,
      children,
      showCloseWarning = false,
      onAttemptClose,
      xButtonRef,
      preventOutsideClose = false,
      ...props
    },
    ref,
  ) => {
    const dismissRef = React.useContext(DialogDismissContext);
    // Publish this content's dismiss preferences so the root `onOpenChange`
    // handler can honour them (escape / outside-press / focus-out).
    if (dismissRef) {
      dismissRef.current = { showCloseWarning, onAttemptClose, preventOutsideClose };
    }

    const handleCloseAttempt = React.useCallback(() => {
      // This handler is now only called when showCloseWarning is true
      if (showCloseWarning) {
        onAttemptClose?.();
      }
    }, [showCloseWarning, onAttemptClose]);

    // Common class names for both button types
    const buttonClassNames =
      "absolute right-4 top-4 rounded-xs opacity-70 ring-offset-background transition-opacity hover:opacity-100 focus:outline-hidden focus:ring-2 focus:ring-ring focus:ring-offset-2 focus:rounded-lg disabled:pointer-events-none data-open:bg-accent text-muted-foreground z-51 [&_svg]:size-[14px] hover:rounded-lg";

    return (
      <DialogPortal>
        <DialogOverlay />
        <DialogPrimitive.Popup
          ref={ref}
          className={cn(
            "fixed left-[50%] top-[50%] z-50 grid w-full max-w-lg translate-x-[-50%] translate-y-[-50%] gap-4 border bg-background p-6 shadow-lg duration-200 transition-[opacity,scale,translate] data-starting-style:opacity-0 data-starting-style:scale-95 data-ending-style:opacity-0 data-ending-style:scale-95 sm:rounded-lg",
            className,
          )}
          onKeyDown={(e) => {
            // Allow keyboard navigation for nested interactive elements
            if (e.key === "ArrowDown" || e.key === "ArrowUp" || e.key === "Enter") {
              // Let these events propagate to nested components like Combobox
              return;
            }
            // Prevent Tab key from closing the dialog
            if (e.key === "Tab") {
              e.stopPropagation();
            }
          }}
          {...props}
        >
          {children}

          {/* Conditionally render the close button */}
          {showCloseWarning ? (
            <button
              ref={xButtonRef} // Attach ref only needed for anchoring the custom popover
              type="button"
              onClick={handleCloseAttempt} // Call attempt handler
              className={buttonClassNames}
              aria-label="Close dialog with confirmation"
            >
              <XMark iconSize="md-medium" />
            </button>
          ) : (
            // Use DialogPrimitive.Close for standard behavior
            <DialogPrimitive.Close
              render={
                <Button
                  size="icon"
                  variant="ghost"
                  type="button"
                  className={buttonClassNames}
                  aria-label="Close dialog"
                >
                  <XMark iconSize="md-medium" />
                </Button>
              }
            />
          )}
        </DialogPrimitive.Popup>
      </DialogPortal>
    );
  },
);
DialogContent.displayName = "DialogContent";

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

const DialogTitle = React.forwardRef<
  React.ComponentRef<typeof DialogPrimitive.Title>,
  DialogPrimitive.Title.Props
>(({ className, ...props }, ref) => (
  <DialogPrimitive.Title
    ref={ref}
    className={cn("text-lg font-semibold leading-none tracking-tight", className)}
    {...props}
  />
));
DialogTitle.displayName = "DialogTitle";

const DialogDescription = React.forwardRef<
  React.ComponentRef<typeof DialogPrimitive.Description>,
  DialogPrimitive.Description.Props
>(({ className, ...props }, ref) => (
  <DialogPrimitive.Description
    ref={ref}
    className={cn("text-sm text-content-subtle", className)}
    {...props}
  />
));
DialogDescription.displayName = "DialogDescription";

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
