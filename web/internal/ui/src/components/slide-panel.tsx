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
        className={cn(
          "fixed bg-gray-1 border border-grayA-4 rounded-xl overflow-hidden z-51",
          "transition-[transform,opacity] duration-300 ease-out",
          "shadow-md",
          side === "right" ? "right-3" : "left-3",
          isOpen
            ? "translate-x-0 opacity-100"
            : side === "right"
              ? "translate-x-full opacity-0"
              : "-translate-x-full opacity-0",
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
    className={cn("flex items-start justify-between border-b border-grayA-4 px-8 py-5", className)}
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
  <div className={cn("border-t border-grayA-4 bg-gray-1 px-8 py-5", className)}>{children}</div>
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
