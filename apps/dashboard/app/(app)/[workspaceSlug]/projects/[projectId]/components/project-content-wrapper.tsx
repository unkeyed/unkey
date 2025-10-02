"use client";

import { cn } from "@unkey/ui/src/lib/utils";
import type { ReactNode } from "react";
import { useProject } from "../layout-provider";

type ProjectContentWrapperProps = {
  children: ReactNode;
  className?: string;
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
};

export function ProjectContentWrapper({
  children,
  className,
  centered = false,
  maxWidth = "960px",
}: ProjectContentWrapperProps) {
  const { isDetailsOpen } = useProject();

  return (
    <div
      className={cn(
        "transition-all duration-300 ease-in-out",
        isDetailsOpen ? "w-[calc(100vw-616px)]" : "w-[calc(100vw-256px)]",
        centered ? "flex justify-center pb-20 px-8" : "flex flex-col",
        className,
      )}
    >
      {centered ? (
        <div className="flex flex-col w-full mt-4 gap-5" style={{ maxWidth }}>
          {children}
        </div>
      ) : (
        children
      )}
    </div>
  );
}
