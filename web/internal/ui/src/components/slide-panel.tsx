"use client";

import { Dialog as DialogPrimitive } from "@base-ui/react/dialog";
import { XMark } from "@unkey/icons";
import * as React from "react";
import { cn } from "../lib/utils";

type SlidePanelBackdrop = "blur" | "dim" | "none";

type SlidePanelRootProps = {
  children: React.ReactNode;
  isOpen: boolean;
  onClose: () => void;
  onExitComplete?: () => void;
  side?: "left" | "right";
  widthClassName?: string;
  className?: string;
  backdrop?: SlidePanelBackdrop;
  fitContent?: boolean;
};

const SlidePanelRoot = ({
  children,
  isOpen,
  onClose,
  onExitComplete,
  side = "right",
  widthClassName = "w-175",
  className,
  backdrop = "blur",
  fitContent = false,
}: SlidePanelRootProps) => (
  <DialogPrimitive.Root
    open={isOpen}
    modal={false}
    disablePointerDismissal
    onOpenChange={(open) => {
      if (!open) {
        onClose();
      }
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
          onClick={onClose}
          className={cn(
            "fixed inset-0 z-50 transition-opacity duration-300 ease-[cubic-bezier(0.4,0,0.2,1)] motion-reduce:transition-none",
            "data-starting-style:opacity-0 data-ending-style:opacity-0",
            backdrop === "blur" ? "bg-background/5 backdrop-blur-[2px]" : "bg-background/20",
          )}
        />
      )}
      <DialogPrimitive.Popup
        data-slide-panel-open=""
        className={cn(
          "[--slide-panel-inset:0.75rem] [--slide-panel-radius:0.75rem]",
          "fixed z-51 flex flex-col p-px shadow-lg",
          "rounded-(--slide-panel-radius) bg-grayA-4",
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
        <div className="flex min-h-0 flex-1 flex-col overflow-hidden rounded-[calc(var(--slide-panel-radius)_-_1px)] bg-background">
          {children}
        </div>
      </DialogPrimitive.Popup>
    </DialogPrimitive.Portal>
  </DialogPrimitive.Root>
);

SlidePanelRoot.displayName = "SlidePanelRoot";

type SlidePanelHeaderProps = {
  children: React.ReactNode;
  className?: string;
};

const SlidePanelHeader = ({ children, className }: SlidePanelHeaderProps) => (
  <div className={cn("flex items-start justify-between px-6 pt-6 pb-2", className)}>{children}</div>
);

SlidePanelHeader.displayName = "SlidePanelHeader";

const SlidePanelTitle = React.forwardRef<
  React.ComponentRef<typeof DialogPrimitive.Title>,
  DialogPrimitive.Title.Props
>(({ className, ...props }, ref) => (
  <DialogPrimitive.Title
    ref={ref}
    className={cn("text-[18px] font-semibold leading-tight tracking-tight text-gray-12", className)}
    {...props}
  />
));

SlidePanelTitle.displayName = "SlidePanelTitle";

const SlidePanelDescription = React.forwardRef<
  React.ComponentRef<typeof DialogPrimitive.Description>,
  DialogPrimitive.Description.Props
>(({ className, ...props }, ref) => (
  <DialogPrimitive.Description
    ref={ref}
    className={cn("text-[13px] leading-5 text-gray-11", className)}
    {...props}
  />
));

SlidePanelDescription.displayName = "SlidePanelDescription";

type SlidePanelContentProps = {
  children: React.ReactNode;
  className?: string;
};

const SlidePanelContent = ({ children, className }: SlidePanelContentProps) => (
  <div className={cn("min-h-0 flex-1", className)}>{children}</div>
);

SlidePanelContent.displayName = "SlidePanelContent";

type SlidePanelFooterProps = {
  children: React.ReactNode;
  className?: string;
};

const SlidePanelFooter = ({ children, className }: SlidePanelFooterProps) => (
  <div className={cn("border-t border-gray-4 px-6 py-3", className)}>{children}</div>
);

SlidePanelFooter.displayName = "SlidePanelFooter";

type SlidePanelCloseButtonProps = DialogPrimitive.Close.Props;

const SlidePanelCloseButton = React.forwardRef<
  React.ComponentRef<typeof DialogPrimitive.Close>,
  SlidePanelCloseButtonProps
>(({ className, ...props }, ref) => (
  <DialogPrimitive.Close
    ref={ref}
    aria-label="Close panel"
    className={cn(
      "inline-flex size-9 shrink-0 cursor-pointer items-center justify-center rounded-md text-gray-10 transition-colors hover:bg-grayA-3 hover:text-gray-12",
      className,
    )}
    {...props}
  >
    <XMark iconSize="lg-medium" />
  </DialogPrimitive.Close>
));

SlidePanelCloseButton.displayName = "SlidePanelCloseButton";

export const SlidePanel = {
  Root: SlidePanelRoot,
  Header: SlidePanelHeader,
  Title: SlidePanelTitle,
  Description: SlidePanelDescription,
  Content: SlidePanelContent,
  Footer: SlidePanelFooter,
  CloseButton: SlidePanelCloseButton,
};

export type {
  SlidePanelBackdrop,
  SlidePanelRootProps,
  SlidePanelHeaderProps,
  SlidePanelContentProps,
  SlidePanelFooterProps,
  SlidePanelCloseButtonProps,
};
