"use client";

import { type CSSProperties, type ReactNode, isValidElement } from "react";
import { cn } from "../lib/utils";
import { HoverCard, HoverCardContent, HoverCardTrigger } from "./hover-card";

type HoverCardPosition = {
  side?: "top" | "right" | "bottom" | "left";
  align?: "start" | "center" | "end";
  sideOffset?: number;
};

type InfoHoverCardProps = {
  content: ReactNode;
  children: ReactNode;
  position?: HoverCardPosition;
  delayDuration?: number;
  asChild?: boolean;
  className?: string;
  style?: CSSProperties;
  triggerClassName?: string;
};

export function InfoHoverCard({
  content,
  children,
  position,
  delayDuration,
  asChild = false,
  className,
  style,
  triggerClassName,
}: InfoHoverCardProps) {
  return (
    <HoverCard>
      {asChild && isValidElement(children) ? (
        <HoverCardTrigger delay={delayDuration} render={children} className={triggerClassName} />
      ) : (
        <HoverCardTrigger delay={delayDuration} render={<span />} className={triggerClassName}>
          {children}
        </HoverCardTrigger>
      )}
      <HoverCardContent
        side={position?.side ?? "right"}
        align={position?.align ?? "center"}
        sideOffset={position?.sideOffset}
        style={style}
        className={cn("w-auto px-3 py-2 text-xs", className)}
      >
        {content}
      </HoverCardContent>
    </HoverCard>
  );
}
