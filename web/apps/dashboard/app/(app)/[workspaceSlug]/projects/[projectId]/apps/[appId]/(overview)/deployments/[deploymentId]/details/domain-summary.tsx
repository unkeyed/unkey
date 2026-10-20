"use client";

import { Skeleton } from "@unkey/ui";
import type { DisplayDomain } from "../../../../components/display-domain";
import { DomainOverflow } from "../../../../components/domain-overflow";

export function DomainSummary({
  primary,
  additional,
  isLoading,
}: { primary: DisplayDomain | null; additional: readonly DisplayDomain[]; isLoading: boolean }) {
  if (isLoading) {
    return <Skeleton className="h-4 w-48 rounded" />;
  }
  if (!primary) {
    return <span className="text-gray-10">No domains yet</span>;
  }
  return (
    <span className="flex min-w-0 items-center gap-2">
      <a
        href={primary.url}
        target="_blank"
        rel="noopener noreferrer"
        className="truncate text-sm font-medium text-gray-12 transition-colors hover:text-gray-11"
      >
        {primary.hostname}
      </a>
      <DomainOverflow domains={additional} />
    </span>
  );
}
