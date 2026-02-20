"use client";

import { ChevronRight } from "@unkey/icons";
import * as React from "react";
import { cn } from "../lib/utils";

export type ChevronState = "hidden" | "interactive" | "disabled";

export type SettingCardBorder = "top" | "middle" | "bottom" | "both" | "none" | "default";

type SettingCardProps = {
  title: string | React.ReactNode;
  description: string | React.ReactNode;
  children?: React.ReactNode;
  className?: string;
  border?: SettingCardBorder;
  contentWidth?: string;
  icon?: React.ReactNode;
  expandable?: React.ReactNode;
  defaultExpanded?: boolean;
  chevronState?: ChevronState;
};

const SettingCardGroupContext = React.createContext(false);

function SettingCardGroup({ children }: { children: React.ReactNode }) {
  return (
    <SettingCardGroupContext.Provider value={true}>
      <div className="border border-grayA-4 rounded-xl overflow-hidden divide-y divide-grayA-4">
        {children}
      </div>
    </SettingCardGroupContext.Provider>
  );
}
SettingCardGroup.displayName = "SettingCardGroup";

function SettingCard({
  title,
  description,
  children,
  className,
  border = "default",
  contentWidth = "w-[420px]",
  icon,
  expandable,
  defaultExpanded = false,
  chevronState,
}: SettingCardProps) {
  const [isExpanded, setIsExpanded] = React.useState(defaultExpanded);
  const contentRef = React.useRef<HTMLDivElement>(null);
  const innerRef = React.useRef<HTMLDivElement>(null);
  const [contentHeight, setContentHeight] = React.useState(0);
  const inGroup = React.useContext(SettingCardGroupContext);

  React.useEffect(() => {
    const inner = innerRef.current;
    if (!inner) {
      return;
    }
    const observer = new ResizeObserver((entries) => {
      for (const entry of entries) {
        setContentHeight(entry.borderBoxSize[0].blockSize);
      }
    });
    observer.observe(inner);
    return () => observer.disconnect();
  }, []);

  // Determine effective chevron state
  const effectiveChevronState: ChevronState =
    chevronState ?? (expandable ? "interactive" : "hidden");

  const shouldShowChevron = effectiveChevronState !== "hidden";
  const isInteractive = effectiveChevronState === "interactive" && expandable;

  const getBorderRadiusClass = () => {
    if (inGroup) {
      return "";
    }
    if (border === "none" || border === "default") {
      return "";
    }
    if (border === "top") {
      return "rounded-t-xl";
    }
    if (border === "bottom") {
      return !expandable || !isExpanded ? "rounded-b-xl" : "";
    }
    if (border === "both") {
      const bottom = !expandable || !isExpanded ? "rounded-b-xl" : "";
      return cn("rounded-t-xl", bottom);
    }
    return "";
  };

  const borderClass = inGroup
    ? {}
    : {
        "border border-grayA-4": border !== "none",
        "border-t-0": border === "bottom",
        "border-b-0": border === "top",
      };

  const expandedBottomRadius =
    !inGroup && expandable && isExpanded && (border === "bottom" || border === "both")
      ? "rounded-b-xl"
      : "";

  const handleToggle = () => {
    if (!isInteractive) return;
    setIsExpanded((prev) => {
      if (!prev) {
        contentRef.current?.addEventListener(
          "transitionend",
          () => {
            const inner = innerRef.current;
            if (!inner) {
              return;
            }
            const overflow = inner.getBoundingClientRect().bottom - window.innerHeight;
            if (overflow > 0)
              findScrollParent(inner).scrollBy({ top: overflow + 16, behavior: "smooth" });
          },
          { once: true },
        );
      }
      return !prev;
    });
  };

  return (
    <div className={cn(getBorderRadiusClass(), borderClass, expandedBottomRadius)}>
      <div
        className={cn(
          "px-6 py-6 lg:w-full flex gap-6 lg:justify-between lg:items-center flex-col lg:flex-row group",
          isInteractive && "cursor-pointer",
          className,
        )}
        onKeyDown={(e) => {
          if (!isInteractive) {
            return;
          }
          if (e.key === "Enter") {
            e.preventDefault();
            handleToggle();
          }
        }}
        onClick={isInteractive ? handleToggle : undefined}
      >
        <div className="flex gap-4 items-center">
          {icon && (
            <div className="bg-gray-3 size-8 rounded-[10px] flex items-center justify-center">
              {icon}
            </div>
          )}
          <div className="flex flex-col gap-1 text-sm w-fit">
            <div className="font-medium text-gray-12 text-[13px] leading-4 tracking-normal">
              {title}
            </div>
            <div className="font-normal text-gray-9 text-xs leading-4 tracking-normal">
              {description}
            </div>
          </div>
        </div>
        <div className={cn("flex w-full items-center gap-4", contentWidth)}>
          {children}
          {shouldShowChevron && (
            <ChevronRight
              className={cn(
                "text-gray-10 transition-all duration-300 ease-out flex-shrink-0",
                isExpanded && "rotate-90",
                effectiveChevronState !== "disabled" && "group-hover:text-gray-11",
                effectiveChevronState === "disabled" && "opacity-40 cursor-not-allowed",
              )}
              iconSize="sm-medium"
            />
          )}
        </div>
      </div>
      {expandable && (
        <div
          ref={contentRef}
          className="overflow-hidden transition-all duration-300 ease-out"
          style={{
            maxHeight: isExpanded ? `${contentHeight}px` : "0px",
          }}
        >
          <div
            ref={innerRef}
            className={cn(
              "border-t border-grayA-4 transition-all duration-300 ease-out",
              isExpanded ? "opacity-100 translate-y-0 delay-75" : "opacity-0 -translate-y-2",
            )}
          >
            {expandable}
          </div>
        </div>
      )}
    </div>
  );
}

// Source - https://stackoverflow.com/a/78682259
// Posted by dgropp
const scrollStyles = ["scroll", "auto"];
function findScrollParent(element: HTMLElement | null): HTMLElement | Window {
  const parent = element?.parentElement;
  if (!parent) return window;
  const { overflowY } = getComputedStyle(parent);
  if (scrollStyles.includes(overflowY)) return parent;
  return findScrollParent(parent);
}

SettingCard.displayName = "SettingCard";

export { SettingCard, SettingCardGroup };
