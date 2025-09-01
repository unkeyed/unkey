import { cn } from "@unkey/ui/src/lib/utils";
import type { ReactNode } from "react";

export function Card({
  children,
  className,
}: {
  children: ReactNode;
  className?: string;
}) {
  return <div className={cn("border border-gray-4 w-full", className)}>{children}</div>;
}
