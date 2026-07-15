import type React from "react";
import { cn } from "../lib/utils";

export function FullScreenLayout({
  className,
  children,
  ...props
}: React.HTMLAttributes<HTMLDivElement>) {
  return (
    <div
      className={cn("relative flex min-h-dvh w-full flex-col items-center", className)}
      {...props}
    >
      {children}
    </div>
  );
}

export function FullScreenContent({
  className,
  children,
  ...props
}: React.HTMLAttributes<HTMLDivElement>) {
  return (
    <div
      className={cn("flex w-full flex-1 flex-col items-center justify-center", className)}
      {...props}
    >
      {children}
    </div>
  );
}
