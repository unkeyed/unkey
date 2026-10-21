"use client";

import {
  LOG_OUTCOME_DEFINITIONS,
  getStatusType,
} from "@/components/key-details-logs-table/utils/outcome-definitions";
import { LogSection } from "@/components/logs/details/log-details/components/log-section";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { routes } from "@/lib/navigation/routes";
import { trpc } from "@/lib/trpc/client";
import type { RequestLogsResponse } from "@unkey/clickhouse/src/frontline";
import { Badge, CopyButton } from "@unkey/ui";
import Link from "next/link";
import type { ReactNode } from "react";

export function GatewayKeyVerification({ log }: { log: RequestLogsResponse }) {
  const workspace = useWorkspaceNavigation();
  const { data, isLoading, error } = trpc.deploy.requestLogs.keyVerification.useQuery(
    {
      requestId: log.request_id,
      startTime: Math.max(0, log.time - 60_000),
      endTime: log.time + log.total_latency + 60_000,
    },
    {
      retry: false,
      refetchOnWindowFocus: false,
      refetchInterval: (data, query) =>
        !data?.verification && query.state.dataUpdateCount + query.state.errorUpdateCount < 6
          ? 2_000
          : false,
    },
  );

  const verification = data?.verification;
  let details: ReactNode;
  if (isLoading) {
    details = <span className="text-grayA-10">Loading key verification...</span>;
  } else if (verification) {
    const outcome = Object.values(LOG_OUTCOME_DEFINITIONS).find(
      (definition) => definition.type === verification.outcome,
    );
    details = (
      <div className="flex flex-col gap-2">
        <div className="flex items-center justify-between gap-4">
          <span className="text-gray-11">Outcome:</span>
          <Badge variant={outcome?.type ? getStatusType(outcome.type) : "secondary"}>
            {verification.outcome || "Unknown"}
          </Badge>
        </div>
        {verification.key?.name && (
          <div className="flex items-center justify-between gap-4">
            <span className="text-gray-11 shrink-0">Key name:</span>
            <span className="truncate" title={verification.key.name}>
              {verification.key.name}
            </span>
          </div>
        )}
        <div className="flex items-center justify-between gap-4">
          <span className="text-gray-11 shrink-0">Key ID:</span>
          <div className="flex items-center gap-2 min-w-0">
            {verification.key ? (
              <Link
                className="font-mono underline decoration-dotted truncate"
                title={`View details for ${verification.key_id}`}
                href={routes.apis.keys.detail({
                  workspaceSlug: workspace.slug,
                  projectId: verification.key.projectId,
                  apiId: verification.key.apiId,
                  keyAuthId: verification.key_space_id,
                  keyId: verification.key_id,
                })}
              >
                {verification.key_id}
              </Link>
            ) : (
              <span className="font-mono truncate">{verification.key_id || "Unavailable"}</span>
            )}
            {verification.key_id && (
              <CopyButton value={verification.key_id} variant="ghost" className="h-4 w-4" />
            )}
          </div>
        </div>
      </div>
    );
  } else {
    details = (
      <span className="text-grayA-10">
        {error
          ? "Could not load key verification."
          : "Key verification unavailable for this request."}
      </span>
    );
  }

  return <LogSection title="Key verification" details={details} />;
}
