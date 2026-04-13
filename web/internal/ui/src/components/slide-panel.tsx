"use client";

import * as React from "react";
import { createPortal } from "react-dom";
import { createContext } from "../lib/create-context";
import { cn } from "../lib/utils";

type SlidePanelContextValue = {
  isOpen: boolean;
  onClose: () => void;
};

const [SlidePanelProvider, useSlidePanelContext] =
  createContext<SlidePanelContextValue>("SlidePanel");

type SlidePanelRootProps = {
  children: React.ReactNode;
  isOpen: boolean;
  onClose: () => void;
  side?: "left" | "right";
  topOffset?: number;
  widthClassName?: string;
  className?: string;
};

const SlidePanelRoot = ({
  children,
  isOpen,
  onClose,
  side = "right",
  topOffset = 0,
  widthClassName = "w-175",
  className,
}: SlidePanelRootProps) => {
  React.useEffect(
    function closeOnEscape() {
      if (!isOpen) {
        return;
      }
      const handler = (e: KeyboardEvent) => {
        if (e.key === "Escape") {
          onClose();
        }
      };
      document.addEventListener("keydown", handler);
      return () => document.removeEventListener("keydown", handler);
    },
    [isOpen, onClose],
  );

  React.useEffect(
    function markBodyWhileOpen() {
      if (!isOpen) {
        return;
      }
      // Signal to global CSS that a SlidePanel is open so portaled
      // floating UI (tooltips) can be lifted above the panel.
      // Reference-counted to handle nested/sibling panels correctly.
      const prev = document.body.dataset.slidePanelOpen;
      const count = prev ? Number.parseInt(prev, 10) + 1 : 1;
      document.body.dataset.slidePanelOpen = String(count);
      return () => {
        const next = Number.parseInt(document.body.dataset.slidePanelOpen ?? "1", 10) - 1;
        if (next <= 0) {
          delete document.body.dataset.slidePanelOpen;
        } else {
          document.body.dataset.slidePanelOpen = String(next);
        }
      };
    },
    [isOpen],
  );

  const panel = (
    <SlidePanelProvider isOpen={isOpen} onClose={onClose}>
      {/* Backdrop 
         biome-ignore lint/a11y/useKeyWithClickEvents: Safe to leave
        */}
      <div
        className={cn(
          "fixed inset-0 z-50 bg-background/5 transition-opacity duration-300",
          isOpen
            ? "opacity-100 backdrop-blur-[2px]"
            : "opacity-0 pointer-events-none backdrop-blur-none",
        )}
        onClick={onClose}
        aria-hidden="true"
      />
      {/* Panel */}
      <div
        aria-hidden={!isOpen}
        inert={!isOpen || undefined}
        className={cn(
          "fixed dark:bg-black bg-white border border-gray-4 rounded-xl overflow-hidden z-51",
          "transition-transform duration-300 ease-out",
          "shadow-md",
          side === "right" ? "right-3" : "left-3",
          isOpen
            ? "translate-x-0"
            : side === "right"
              ? "translate-x-[calc(100%+0.75rem)]"
              : "-translate-x-[calc(100%+0.75rem)]",
          widthClassName,
          className,
        )}
        style={{
          top: `${topOffset + 12}px`,
          height: `calc(100vh - ${topOffset + 24}px)`,
          willChange: isOpen ? "transform, opacity" : "auto",
        }}
      >
        <div className="h-full flex flex-col">{children}</div>
      </div>
    </SlidePanelProvider>
  );

  return createPortal(panel, document.body);
};

SlidePanelRoot.displayName = "SlidePanelRoot";

type SlidePanelHeaderProps = {
  children: React.ReactNode;
  className?: string;
};

const SlidePanelHeader = ({ children, className }: SlidePanelHeaderProps) => (
  <div
    className={cn(
      "flex items-start justify-between border-b border-gray-4 px-8 py-5 bg-white dark:bg-black ",
      className,
    )}
  >
    {children}
  </div>
);

SlidePanelHeader.displayName = "SlidePanelHeader";

type SlidePanelContentProps = {
  children: React.ReactNode;
  className?: string;
  stagger?: boolean;
  staggerDelay?: number;
};

const SlidePanelContent = ({
  children,
  className,
  stagger = true,
  staggerDelay = 150,
}: SlidePanelContentProps) => {
  const { isOpen } = useSlidePanelContext("SlidePanelContent");

  return (
    <div
      className={cn(
        "flex-1 min-h-0",
        stagger && "transition-[transform,opacity] duration-500 ease-out",
        stagger && (isOpen ? "translate-x-0 opacity-100" : "translate-x-6 opacity-0"),
        className,
      )}
      style={stagger ? { transitionDelay: isOpen ? `${staggerDelay}ms` : "0ms" } : undefined}
    >
      {children}
    </div>
  );
};

SlidePanelContent.displayName = "SlidePanelContent";

type SlidePanelFooterProps = {
  children: React.ReactNode;
  className?: string;
};

const SlidePanelFooter = ({ children, className }: SlidePanelFooterProps) => (
  <div className={cn("bg-white dark:bg-black border-t border-gray-4 px-8 py-5", className)}>
    {children}
  </div>
);

SlidePanelFooter.displayName = "SlidePanelFooter";

type SlidePanelCloseProps = React.ComponentPropsWithoutRef<"button">;

const SlidePanelClose = React.forwardRef<HTMLButtonElement, SlidePanelCloseProps>(
  ({ onClick, ...props }, ref) => {
    const { onClose } = useSlidePanelContext("SlidePanelClose");

    const handleClick = React.useCallback(
      (e: React.MouseEvent<HTMLButtonElement>) => {
        onClick?.(e);
        if (!e.defaultPrevented) {
          onClose();
        }
      },
      [onClick, onClose],
    );

    return <button ref={ref} type="button" {...props} onClick={handleClick} />;
  },
);

SlidePanelClose.displayName = "SlidePanelClose";

export const SlidePanel = Object.assign(
  {},
  {
    Root: SlidePanelRoot,
    Header: SlidePanelHeader,
    Content: SlidePanelContent,
    Footer: SlidePanelFooter,
    Close: SlidePanelClose,
  },
);

export type {
  SlidePanelRootProps,
  SlidePanelHeaderProps,
  SlidePanelContentProps,
  SlidePanelFooterProps,
  SlidePanelCloseProps,
};
