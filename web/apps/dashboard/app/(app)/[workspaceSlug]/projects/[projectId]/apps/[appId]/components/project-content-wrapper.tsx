"use client";

import { cn } from "@unkey/ui/src/lib/utils";
import type { PropsWithChildren } from "react";

type ProjectContentWrapperProps = PropsWithChildren<{
  className?: React.ComponentProps<"div">["className"];
  /**
   * If true, centers content and applies max-width constraint
   * @default false
   */
  centered?: boolean;
  /**
   * Max width for centered content
   * @default "960px"
   */
  maxWidth?: string;
}>;

export function ProjectContentWrapper({
  children,
  className,
  centered = false,
  maxWidth = "960px",
}: ProjectContentWrapperProps) {
  return (
    <div
      className={cn(
        "w-full",
        centered ? "flex justify-center pb-20 px-8" : "flex flex-col",
        className,
      )}
    >
      {centered ? (
        <div className="flex flex-col w-full mt-6 gap-5" style={{ maxWidth }}>
          {children}
        </div>
      ) : (
        children
      )}
    </div>
  );
}
