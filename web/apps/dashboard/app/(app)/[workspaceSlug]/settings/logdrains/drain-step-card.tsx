"use client";

import { Button, cn } from "@unkey/ui";
import type { ReactNode } from "react";

export function DrainStepCard({
  state,
  step,
  title,
  icon,
  onReset,
  children,
  footer,
}: {
  state: "waiting" | "active" | "settled";
  step: number;
  title: string;
  icon?: ReactNode;
  onReset?: () => void;
  children?: ReactNode;
  footer?: ReactNode;
}) {
  return (
    <div className="overflow-hidden rounded-lg border border-grayA-4 bg-background">
      <div
        className={cn(
          "flex w-full items-center justify-between gap-3 px-4 py-4",
          state === "waiting" && "opacity-50",
        )}
      >
        <div className="flex min-w-0 items-center gap-3">
          {state === "settled" && icon ? (
            <span className="flex size-6 shrink-0 items-center justify-center rounded-md text-gray-12 ring-1 ring-grayA-4">
              {icon}
            </span>
          ) : (
            <span
              className={cn(
                "flex size-6 shrink-0 items-center justify-center rounded-full text-[12px] font-medium tabular-nums",
                state === "waiting" ? "bg-grayA-2 text-gray-9" : "bg-grayA-3 text-gray-12",
              )}
            >
              {step}
            </span>
          )}
          <span
            className={cn(
              "truncate text-[14px] font-semibold",
              state === "waiting" ? "text-gray-9" : "text-accent-12",
            )}
          >
            {title}
          </span>
        </div>
        {state === "settled" && onReset ? (
          <Button type="button" variant="outline" size="sm" onClick={onReset}>
            Change
          </Button>
        ) : null}
      </div>

      {state === "active" && children ? (
        <div className="duration-200 ease-out animate-in fade-in motion-reduce:animate-none">
          <div className="px-4 pb-5">{children}</div>
          {footer ? (
            <div className="border-t border-gray-4 bg-grayA-2 px-4 py-4">{footer}</div>
          ) : null}
        </div>
      ) : null}
    </div>
  );
}
