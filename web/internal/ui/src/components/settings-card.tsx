"use client";

import { ChevronRight } from "@unkey/icons";
import * as React from "react";
import { cn } from "../lib/utils";
import { Button } from "./buttons/button";
import { InfoTooltip } from "./info-tooltip";

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
  iconClassName?: string;
  expandable?: React.ReactNode;
  defaultExpanded?: boolean;
  chevronState?: ChevronState;
  truncateDescription?: boolean;
};

const SettingCardGroupContext = React.createContext(false);

function SettingCardGroup({ children }: { children: React.ReactNode }) {
  return (
    <SettingCardGroupContext.Provider value={true}>
      <div className="border border-grayA-4 rounded-[14px] overflow-hidden divide-y divide-grayA-4">
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
  iconClassName,
  expandable,
  defaultExpanded = false,
  chevronState,
  truncateDescription = false,
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
      return "rounded-t-[14px]";
    }
    if (border === "bottom") {
      return !expandable || !isExpanded ? "rounded-b-[14px]" : "";
    }
    if (border === "both") {
      const bottom = !expandable || !isExpanded ? "rounded-b-[14px]" : "";
      return cn("rounded-t-[14px]", bottom);
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
      ? "rounded-b-[14px]"
      : "";

  const handleToggle = () => {
    if (!isInteractive) {
      return;
    }
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
            if (overflow > 0) {
              findScrollParent(inner).scrollBy({ top: overflow + 16, behavior: "smooth" });
            }
          },
          { once: true },
        );
      }
      return !prev;
    });
  };

  return (
    <div className={cn("w-full", getBorderRadiusClass(), borderClass, expandedBottomRadius)}>
      <div
        className={cn(
          "px-4 py-[18px] lg:w-full flex gap-6 lg:justify-between lg:items-center flex-col lg:flex-row group",
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
            <div
              className={cn(
                "bg-gray-3 size-8 rounded-[10px] flex items-center justify-center shrink-0 dark:ring-1 dark:ring-gray-4 dark:shadow-none shadow-sm shadow-grayA-8/20",
                iconClassName,
              )}
            >
              {icon}
            </div>
          )}
          <div className="flex flex-col gap-1 text-sm w-fit">
            <div className="font-medium text-gray-12 text-[13px] leading-4 tracking-normal">
              {title}
            </div>
            <InfoTooltip
              asChild
              content={
                <div className="whitespace-pre-wrap text-xs break-all max-w-xs">{description}</div>
              }
              position={{ side: "bottom", align: "start" }}
              disabled={!truncateDescription}
            >
              <div
                className={cn(
                  "font-normal text-gray-11 text-xs leading-4 tracking-normal max-w-[600px]",
                  truncateDescription && "truncate",
                )}
              >
                {description}
              </div>
            </InfoTooltip>
          </div>
        </div>
        <div className={cn("flex w-full items-center gap-4", contentWidth)}>
          {children}
          {shouldShowChevron && (
            <ChevronRight
              className={cn(
                "text-gray-10 transition-all duration-300 ease-out shrink-0",
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
  if (!parent) {
    return window;
  }
  const { overflowY } = getComputedStyle(parent);
  if (scrollStyles.includes(overflowY)) {
    return parent;
  }
  return findScrollParent(parent);
}

SettingCard.displayName = "SettingCard";

type SettingsZoneVariant = "danger" | "warning";

const SettingsZoneContext = React.createContext<SettingsZoneVariant>("danger");

const zoneStyles: Record<SettingsZoneVariant, { heading: string; border: string }> = {
  danger: { heading: "text-error-11", border: "border-error-7 divide-error-7" },
  warning: { heading: "text-warning-11", border: "border-warning-7 divide-warning-7" },
};

function SettingsZone({
  children,
  className,
  variant,
  title,
}: {
  children: React.ReactNode;
  className?: string;
  variant: SettingsZoneVariant;
  title: string;
}) {
  const styles = zoneStyles[variant];
  return (
    <SettingsZoneContext.Provider value={variant}>
      <div className={cn("w-full", className)}>
        <h2 className={cn("font-semibold text-lg mb-4", styles.heading)}>{title}</h2>
        <div className={cn("rounded-lg border overflow-hidden divide-y", styles.border)}>
          {children}
        </div>
      </div>
    </SettingsZoneContext.Provider>
  );
}

SettingsZone.displayName = "SettingsZone";

function SettingsDangerZone({
  children,
  className,
}: { children: React.ReactNode; className?: string }) {
  return (
    <SettingsZone variant="danger" title="Danger Zone" className={className}>
      {children}
    </SettingsZone>
  );
}

SettingsDangerZone.displayName = "SettingsDangerZone";

type SettingsZoneAction = {
  label: string;
  onClick: () => void;
  loading?: boolean;
  disabled?: boolean;
  className?: string;
};

const zoneButtonProps: Record<
  SettingsZoneVariant,
  { variant: "destructive" | "primary"; color?: "danger" | "warning" }
> = {
  danger: { variant: "destructive" },
  warning: { variant: "primary", color: "warning" },
};

function SettingsZoneRow({
  title,
  description,
  action,
}: {
  title: string;
  description: React.ReactNode;
  action: SettingsZoneAction;
}) {
  const zoneVariant = React.useContext(SettingsZoneContext);
  const btnProps = zoneButtonProps[zoneVariant];

  return (
    <div className="flex items-center justify-between px-4 py-5">
      <div>
        <p className="font-medium text-gray-12 text-sm">{title}</p>
        <p className="text-gray-11 text-[13px]">{description}</p>
      </div>
      <Button
        variant={btnProps.variant}
        color={btnProps.color}
        size="md"
        className={cn("shrink-0 px-3", action.className)}
        loading={action.loading}
        disabled={action.disabled}
        onClick={action.onClick}
      >
        {action.label}
      </Button>
    </div>
  );
}

SettingsZoneRow.displayName = "SettingsZoneRow";

export { SettingCard, SettingCardGroup, SettingsDangerZone, SettingsZone, SettingsZoneRow };
