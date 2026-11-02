import type * as React from "react";
import { cn } from "../lib/utils";

export function DetailList({ className, ...props }: React.ComponentProps<"dl">) {
  return (
    <dl
      data-slot="detail-list"
      className={cn(
        "flex flex-col divide-y divide-grayA-4 rounded-lg border border-grayA-4",
        className,
      )}
      {...props}
    />
  );
}

export function DetailRow({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div
      data-slot="detail-row"
      className="grid grid-cols-[112px_minmax(0,1fr)] items-start gap-4 px-4 py-3 text-sm"
    >
      <dt className="font-medium text-gray-10">{label}</dt>
      <dd className="min-w-0 text-gray-12">{children}</dd>
    </div>
  );
}
