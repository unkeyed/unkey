"use client";

import { trpc } from "@/lib/trpc/client";

export function AlertsNavBadge() {
  const summary = trpc.alerts.summary.useQuery(undefined, {
    staleTime: 60_000,
    refetchInterval: 60_000,
  });
  const count = summary.data?.open ?? 0;

  if (count === 0) {
    return null;
  }

  return (
    <span
      className="inline-flex min-w-5 items-center justify-center rounded-full bg-errorA-3 px-1.5 py-0.5 text-[10px] font-semibold leading-none text-error-11"
      aria-label={`${count} open alerts`}
    >
      {count > 99 ? "99+" : count}
    </span>
  );
}
