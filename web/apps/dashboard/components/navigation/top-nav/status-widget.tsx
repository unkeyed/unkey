"use client";

import { cn } from "@/lib/utils";
import { useQuery } from "@tanstack/react-query";
import { forwardRef } from "react";

const STATUS_PAGE_URL = "https://status.unkey.com";
const SUMMARY_URL = `${STATUS_PAGE_URL}/api/v1/summary`;

type Incident = { id: number | string; name: string };

// Subset of the incident.io status page summary payload we rely on.
// https://status.unkey.com/api/v1/summary
type StatusSummary = {
  ongoing_incidents: Incident[];
  in_progress_maintenances: Incident[];
  scheduled_maintenances: Incident[];
};

type SystemStatus = "operational" | "incident" | "maintenance" | "unknown";

const STATUS_META: Record<SystemStatus, { label: string; dot: string; pulse: boolean }> = {
  operational: { label: "Fully operational", dot: "bg-success-9", pulse: false },
  incident: { label: "Active incident", dot: "bg-error-9", pulse: true },
  maintenance: { label: "Under maintenance", dot: "bg-warning-9", pulse: true },
  unknown: { label: "Status unavailable", dot: "bg-gray-7", pulse: false },
};

function deriveStatus(summary: StatusSummary | undefined): SystemStatus {
  if (!summary) {
    return "unknown";
  }
  if (summary.ongoing_incidents?.length > 0) {
    return "incident";
  }
  if (summary.in_progress_maintenances?.length > 0) {
    return "maintenance";
  }
  return "operational";
}

type StatusWidgetProps = React.ComponentPropsWithoutRef<"a">;

/**
 * StatusWidget shows the live operational status from status.unkey.com and links to it.
 * It is self-contained (fetches client-side, no props required) so it can be dropped anywhere
 * inside the dashboard that lives under the React Query provider. It forwards its ref and props
 * to the underlying anchor, so it composes with primitives like `DropdownMenuItem asChild`.
 */
export const StatusWidget = forwardRef<HTMLAnchorElement, StatusWidgetProps>(function StatusWidget(
  { className, ...props },
  ref,
) {
  const { data, isLoading, isError } = useQuery({
    queryKey: ["status-page-summary"],
    queryFn: async (): Promise<StatusSummary> => {
      const res = await fetch(SUMMARY_URL, { headers: { Accept: "application/json" } });
      if (!res.ok) {
        throw new Error(`status page responded with ${res.status}`);
      }
      return res.json();
    },
    refetchInterval: 5 * 60 * 1000,
    staleTime: 60 * 1000,
    refetchOnWindowFocus: false,
    retry: 1,
  });

  const loading = isLoading && !data;
  const status: SystemStatus = isError ? "unknown" : deriveStatus(data);
  const meta = STATUS_META[status];

  return (
    <a
      ref={ref}
      href={STATUS_PAGE_URL}
      target="_blank"
      rel="noreferrer"
      aria-label={`${meta.label}. View status page`}
      className={cn(
        "group/status flex w-full items-center gap-3 text-sm font-medium text-accent-12",
        className,
      )}
      {...props}
    >
      <span className="relative flex size-4 shrink-0 items-center justify-center">
        {meta.pulse && !loading ? (
          <span
            className={cn(
              "absolute inline-flex size-2 animate-ping rounded-full opacity-75",
              meta.dot,
            )}
          />
        ) : null}
        <span
          className={cn(
            "relative inline-flex size-2 rounded-full",
            loading ? "animate-pulse bg-gray-7" : meta.dot,
          )}
        />
      </span>
      <span className="truncate">{loading ? "Checking status" : meta.label}</span>
    </a>
  );
});
