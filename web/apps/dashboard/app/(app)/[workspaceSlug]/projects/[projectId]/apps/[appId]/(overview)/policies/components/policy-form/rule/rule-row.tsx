import { cn } from "cn";
import type { ReactNode } from "react";

export function RuleRow({ className, children }: { className?: string; children: ReactNode }) {
  return (
    <div className={cn("rounded-md bg-raised ring-1 ring-grayA-4", className)}>{children}</div>
  );
}

export function RuleLabel({ children }: { children: ReactNode }) {
  return (
    <span className="w-12 shrink-0 font-mono text-2xs font-medium uppercase text-gray-10">
      {children}
    </span>
  );
}
