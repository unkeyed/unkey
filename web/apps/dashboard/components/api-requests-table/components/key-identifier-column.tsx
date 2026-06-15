"use client";
import {
  getErrorPercentage,
  getErrorSeverity,
} from "@/components/api-requests-table/utils/calculate-blocked-percentage";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { routes } from "@/lib/navigation/routes";
import { shortenId } from "@/lib/shorten-id";
import { cn } from "@/lib/utils";
import type { KeysOverviewLog } from "@unkey/clickhouse/src/keys/keys";
import { TriangleWarning2 } from "@unkey/icons";
import { InfoTooltip, Loading } from "@unkey/ui";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useCallback, useState } from "react";

type KeyIdentifierColumnProps = {
  log: KeysOverviewLog;
  apiId: string;
  onNavigate?: () => void;
};

// Get warning icon based on error severity
const getWarningIcon = (severity: string) => {
  switch (severity) {
    case "high":
      return <TriangleWarning2 className="text-error-11" iconSize="md-medium" />;
    case "moderate":
      return <TriangleWarning2 className="text-orange-11" iconSize="md-medium" />;
    case "low":
      return <TriangleWarning2 className="text-warning-11" iconSize="md-medium" />;
    default:
      return <TriangleWarning2 className="invisible" iconSize="md-medium" />;
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
  const workspace = useWorkspaceNavigation();

  const router = useRouter();
  const errorPercentage = getErrorPercentage(log);
  const severity = getErrorSeverity(log);
  const hasErrors = severity !== "none";
  const [isNavigating, setIsNavigating] = useState(false);

  const keyAuthId = log.key_details?.key_auth_id;
  const keyHref = keyAuthId
    ? routes.apis.keys.detail({
        workspaceSlug: workspace.slug,
        apiId,
        keyAuthId,
        keyId: log.key_id,
      })
    : undefined;

  const handleLinkClick = useCallback(
    (e: React.MouseEvent) => {
      e.preventDefault();
      if (!keyHref) {
        return;
      }
      setIsNavigating(true);

      onNavigate?.();

      router.push(keyHref);
    },
    [keyHref, onNavigate, router],
  );

  return (
    <div className="flex gap-6 items-center pl-2">
      <InfoTooltip
        variant="inverted"
        content={<p className="text-xs">{getWarningMessage(severity, errorPercentage)}</p>}
        position={{ side: "right", align: "center" }}
      >
        {isNavigating ? (
          <div className="size-3 items-center justify-center flex">
            <Loading size={18} />
          </div>
        ) : (
          <div className={cn("transition-opacity", hasErrors ? "opacity-100" : "opacity-0")}>
            {getWarningIcon(severity)}
          </div>
        )}
      </InfoTooltip>
      {keyHref ? (
        <Link
          title={`View details for ${log.key_id}`}
          className="font-mono group-hover:underline decoration-dotted"
          href={keyHref}
          onClick={handleLinkClick}
        >
          <div className="font-mono font-medium truncate flex items-center">
            {shortenId(log.key_id)}
          </div>
        </Link>
      ) : (
        <div
          title={`${log.key_id} (deleted)`}
          className="font-mono font-medium truncate flex items-center text-grayA-9"
        >
          {shortenId(log.key_id)}
        </div>
      )}
    </div>
  );
};
