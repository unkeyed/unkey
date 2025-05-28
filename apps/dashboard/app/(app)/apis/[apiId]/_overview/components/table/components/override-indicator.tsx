"use client";
import { RatelimitOverviewTooltip } from "@/app/(app)/ratelimits/[namespaceId]/_overview/components/table/components/ratelimit-overview-tooltip";
import { cn } from "@/lib/utils";
import type { KeysOverviewLog } from "@unkey/clickhouse/src/keys/keys";
import { TriangleWarning2 } from "@unkey/icons";
import { AnimatedLoadingSpinner } from "@unkey/ui";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useCallback, useState } from "react";
import { getErrorPercentage, getErrorSeverity } from "../utils/calculate-blocked-percentage";

type KeyIdentifierColumnProps = {
  log: KeysOverviewLog;
  apiId: string;
  onNavigate?: () => void;
};

// Get warning icon based on error severity
const getWarningIcon = (severity: string) => {
  switch (severity) {
    case "high":
      return <TriangleWarning2 className="text-error-11" />;
    case "moderate":
      return <TriangleWarning2 className="text-orange-11" />;
    case "low":
      return <TriangleWarning2 className="text-warning-11" />;
    default:
      return <TriangleWarning2 className="invisible" />;
  }
};

// Get tooltip message based on error severity
const getWarningMessage = (severity: string, errorRate: number) => {
  switch (severity) {
    case "high":
      return `Critical: ${Math.round(errorRate)}% of requests have failed`;
    case "moderate":
      return `Warning: ${Math.round(errorRate)}% of requests have failed`;
    case "low":
      return `${Math.round(errorRate)}% of requests have been invalid`;
    default:
      return "All requests are valid";
  }
};

export const KeyIdentifierColumn = ({ log, apiId, onNavigate }: KeyIdentifierColumnProps) => {
  const router = useRouter();
  const errorPercentage = getErrorPercentage(log);
  const severity = getErrorSeverity(log);
  const hasErrors = severity !== "none";

  const [isNavigating, setIsNavigating] = useState(false);

  const handleLinkClick = useCallback(
    (e: React.MouseEvent) => {
      e.preventDefault();
      setIsNavigating(true);

      onNavigate?.();

      router.push(`/apis/${apiId}/keys/${log.key_details?.key_auth_id}/${log.key_id}`);
    },
    [apiId, log.key_id, log.key_details?.key_auth_id, onNavigate, router.push],
  );

  return (
    <div className="flex gap-6 items-center pl-2">
      <RatelimitOverviewTooltip
        content={<p className="text-xs">{getWarningMessage(severity, errorPercentage)}</p>}
      >
        {isNavigating ? (
          <div className="size-[12px] items-center justify-center flex">
            <AnimatedLoadingSpinner />
          </div>
        ) : (
          <div className={cn("transition-opacity", hasErrors ? "opacity-100" : "opacity-0")}>
            {getWarningIcon(severity)}
          </div>
        )}
      </RatelimitOverviewTooltip>
      <Link
        title={`View details for ${log.key_id}`}
        className="font-mono group-hover:underline decoration-dotted"
        href={`/apis/${apiId}/keys/${log.key_details?.key_auth_id}/${log.key_id}`}
        onClick={handleLinkClick}
      >
        <div className="font-mono font-medium truncate flex items-center">
          {log.key_id.substring(0, 8)}...
          {log.key_id.substring(log.key_id.length - 4)}
        </div>
      </Link>
    </div>
  );
};
