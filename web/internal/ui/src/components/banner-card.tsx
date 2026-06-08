import { XMark } from "@unkey/icons";
import type * as React from "react";

import { cn } from "../lib/utils";
import { Button } from "./buttons/button";
import { Card, CardContent } from "./card";

export interface BannerCardProps extends React.HTMLAttributes<HTMLDivElement> {
  onDismiss?: () => void;
  illustration?: React.ReactNode;
  dismissLabel?: string;
  ref?: React.Ref<HTMLDivElement>;
}

export function BannerCard({
  onDismiss,
  illustration,
  dismissLabel = "Dismiss",
  className,
  children,
  ref,
  ...props
}: BannerCardProps) {
  return (
    <Card
      ref={ref}
      className={cn("relative overflow-hidden rounded-xl shadow-lg", className)}
      {...props}
    >
      {illustration ? (
        <div className="pointer-events-none absolute inset-0 z-0 overflow-hidden">
          {illustration}
        </div>
      ) : null}
      <CardContent className="relative z-10 p-6">
        {onDismiss ? (
          <Button
            type="button"
            variant="ghost"
            size="icon"
            aria-label={dismissLabel}
            onClick={onDismiss}
            className="absolute right-2 top-2"
          >
            <XMark />
          </Button>
        ) : null}
        {children}
      </CardContent>
    </Card>
  );
}
