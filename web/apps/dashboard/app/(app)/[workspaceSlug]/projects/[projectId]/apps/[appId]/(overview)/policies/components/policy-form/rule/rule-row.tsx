import type { ReactNode } from "react";

export const ROW = "rounded-md bg-raised ring-1 ring-grayA-4";

export function RuleLabel({ children }: { children: ReactNode }) {
  return (
    <span className="w-12 shrink-0 font-mono text-2xs font-medium uppercase text-gray-10">
      {children}
    </span>
  );
}
