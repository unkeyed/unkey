"use client";
import { safeParseJson } from "@/app/(app)/[workspaceSlug]/logs/utils";
import { EMPTY_TEXT, LogDetails } from "@/components/logs/details/log-details";
import { LogSection } from "@/components/logs/details/log-details/components/log-section";
import { shortenId } from "@/lib/shorten-id";
import { cn } from "@/lib/utils";
import { eq, useLiveQuery } from "@tanstack/react-db";
import type { SentinelLogsResponse } from "@unkey/clickhouse/src/sentinel";
import { CodeBranch, CodeCommit, User } from "@unkey/icons";
import { Badge, CopyButton } from "@unkey/ui";
import type React from "react";
import { useProject } from "../../../../layout-provider";
import { useSentinelLogsContext } from "../../../context/sentinel-logs-provider";

type Props = {
  distanceToTop: number;
};

export const SentinelLogDetails = ({ distanceToTop }: Props) => {
  const { setSelectedLog, selectedLog: log } = useSentinelLogsContext();
  const { collections } = useProject();

  const handleClose = () => {
    setSelectedLog(null);
  };

  const { data } = useLiveQuery(
    (q) => {
      return q
        .from({ deployment: collections.deployments })
        .join({ environment: collections.environments }, ({ deployment, environment }) =>
          eq(deployment.environmentId, environment.id),
        )
        .where(({ deployment }) => eq(deployment.id, log?.deployment_id));
    },
    [log?.deployment_id],
  );
  const deployment = data.at(0)?.deployment;
  const environment = data.at(0)?.environment;

  if (!log) {
    // Shouldn't happen
    return null;
  }

  return (
    <LogDetails distanceToTop={distanceToTop} log={log} onClose={handleClose}>
      <LogDetails.Header onClose={handleClose}>
        <SentinelLogHeader log={log} onClose={handleClose} />
      </LogDetails.Header>

      <LogDetails.Section delay={150}>
        <LogSection
          title="Request Header"
          details={log.request_headers.length ? log.request_headers : EMPTY_TEXT}
        />
      </LogDetails.Section>

      <LogDetails.Section delay={200}>
        <LogSection title="Request Body" details={formatRequestBody(log.request_body)} />
      </LogDetails.Section>

      <LogDetails.Section delay={250}>
        <LogSection
          title="Response Header"
          details={log.response_headers.length ? log.response_headers : EMPTY_TEXT}
        />
      </LogDetails.Section>

      <LogDetails.Section delay={300}>
        <LogSection title="Response Body" details={formatResponseBody(log.response_body)} />
      </LogDetails.Section>

      <LogDetails.Section delay={350}>
        <LogSection title="Latency Breakdown" details={formatLatencyMetrics(log)} />
      </LogDetails.Section>

      <LogDetails.Section delay={400}>
        <LogSection
          title="Deployment Information"
          details={formatDeploymentInfo(log, deployment, environment)}
        />
      </LogDetails.Section>

      <LogDetails.Section delay={450}>
        <LogSection title="Meta" details={formatMetaInfo(log)} />
      </LogDetails.Section>

      <LogDetails.Spacer delay={500} />
    </LogDetails>
  );
};

// Custom header for sentinel logs
const SentinelLogHeader = ({
  log,
  onClose,
}: {
  log: SentinelLogsResponse;
  onClose: () => void;
}) => {
  return (
    <div className="border-b-[1px] flex justify-between items-center border-gray-4 h-[45px] px-4 py-2">
      <div className="flex gap-2 items-center min-w-0">
        <Badge className="uppercase px-[6px] rounded-md font-mono bg-accent-3 text-accent-11 hover:bg-accent-4">
          {log.method}
        </Badge>
        <p className="text-xs text-accent-12 truncate flex-1">{log.path}</p>
        <Badge
          className={cn("px-[6px] rounded-md font-mono text-xs", {
            "bg-success-3 text-success-11 hover:bg-success-4":
              log.response_status >= 200 && log.response_status < 300,
            "bg-warning-3 text-warning-11 hover:bg-warning-4":
              log.response_status >= 400 && log.response_status < 500,
            "bg-error-3 text-error-11 hover:bg-error-4": log.response_status >= 500,
          })}
        >
          {log.response_status}
        </Badge>
      </div>
      <div className="flex gap-1 items-center shrink-0">
        <button
          type="button"
          onClick={onClose}
          className="text-grayA-9 hover:text-grayA-11 transition-colors"
          aria-label="Close"
        >
          <svg
            xmlns="http://www.w3.org/2000/svg"
            width="16"
            height="16"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            strokeWidth="2"
            strokeLinecap="round"
            strokeLinejoin="round"
          >
            <line x1="18" y1="6" x2="6" y2="18" />
            <line x1="6" y1="6" x2="18" y2="18" />
          </svg>
        </button>
      </div>
    </div>
  );
};

// Format request body
const formatRequestBody = (body: string): React.ReactNode | string => {
  const parsed = safeParseJson(body);
  return JSON.stringify(parsed, null, 2) === "null" ? (
    <span className="text-xs text-accent-12">{EMPTY_TEXT}</span>
  ) : (
    JSON.stringify(parsed, null, 2)
  );
};

// Format response body
const formatResponseBody = (body: string): React.ReactNode | string => {
  const parsed = safeParseJson(body);
  return JSON.stringify(parsed, null, 2) === "null" ? (
    <span className="text-xs text-accent-12">{EMPTY_TEXT}</span>
  ) : (
    JSON.stringify(parsed, null, 2)
  );
};

// Format latency metrics
const formatLatencyMetrics = (log: SentinelLogsResponse): React.ReactNode => {
  const instancePercent = ((log.instance_latency / log.total_latency) * 100).toFixed(1);
  const sentinelPercent = ((log.sentinel_latency / log.total_latency) * 100).toFixed(1);

  return (
    <div className="flex flex-col gap-2">
      <div className="flex items-center justify-between">
        <span className="text-gray-11">Total Latency:</span>
        <span className="font-mono font-semibold">{log.total_latency}ms</span>
      </div>
      <div className="flex items-center justify-between">
        <span className="text-gray-11">Instance Latency:</span>
        <span className="font-mono">
          {log.instance_latency}ms
          <span className="text-grayA-10 ml-1">({instancePercent}%)</span>
        </span>
      </div>
      <div className="flex items-center justify-between">
        <span className="text-gray-11">Sentinel Latency:</span>
        <span className="font-mono">
          {log.sentinel_latency}ms
          <span className="text-grayA-10 ml-1">({sentinelPercent}%)</span>
        </span>
      </div>
    </div>
  );
};

// Format deployment information
const formatDeploymentInfo = (
  log: SentinelLogsResponse,
  deployment:
    | {
        id: string;
        environmentId: string;
        gitBranch?: string | null;
        gitCommitSha?: string | null;
        gitCommitMessage?: string | null;
        gitCommitAuthorHandle?: string | null;
        gitCommitAuthorAvatarUrl?: string | null;
        status?: string | null;
      }
    | undefined,
  environment: { slug: string } | undefined,
): React.ReactNode => {
  if (!deployment) {
    return (
      <div className="flex flex-col gap-2">
        <div className="flex items-center justify-between">
          <span className="text-gray-11">Deployment ID:</span>
          <div className="flex items-center gap-2">
            <span className="font-mono">{shortenId(log.deployment_id)}</span>
            <CopyButton value={log.deployment_id} variant="ghost" className="h-4 w-4" />
          </div>
        </div>
        <div className="text-xs text-grayA-10 mt-1">(Deployment details not found)</div>
      </div>
    );
  }

  const shortSha = deployment.gitCommitSha?.substring(0, 7);
  const truncatedMessage =
    deployment.gitCommitMessage && deployment.gitCommitMessage.length > 50
      ? `${deployment.gitCommitMessage.substring(0, 50)}...`
      : deployment.gitCommitMessage;

  return (
    <div className="flex flex-col gap-2">
      <div className="flex items-center justify-between">
        <span className="text-gray-11">Deployment ID:</span>
        <div className="flex items-center gap-2">
          <span className="font-mono">{shortenId(log.deployment_id)}</span>
          <CopyButton value={log.deployment_id} variant="ghost" className="h-4 w-4" />
        </div>
      </div>

      {environment && (
        <div className="flex items-center justify-between">
          <span className="text-gray-11">Environment:</span>
          <Badge variant="secondary" className="text-xs">
            {environment.slug}
          </Badge>
        </div>
      )}

      {deployment.gitBranch && (
        <div className="flex items-center justify-between">
          <span className="text-gray-11">Branch:</span>
          <div className="flex items-center gap-1.5">
            <CodeBranch iconSize="sm-regular" className="text-grayA-10 shrink-0" />
            <span className="font-mono truncate max-w-[200px]">{deployment.gitBranch}</span>
          </div>
        </div>
      )}

      {deployment.gitCommitSha && (
        <div className="flex items-center justify-between">
          <span className="text-gray-11">Commit:</span>
          <div className="flex items-center gap-1.5">
            <CodeCommit iconSize="sm-regular" className="text-grayA-10 shrink-0" />
            <span className="font-mono">{shortSha}</span>
            <CopyButton value={deployment.gitCommitSha} variant="ghost" className="h-4 w-4" />
          </div>
        </div>
      )}

      {deployment.gitCommitAuthorHandle && (
        <div className="flex items-center justify-between">
          <span className="text-gray-11">Author:</span>
          <div className="flex items-center gap-1.5">
            {deployment.gitCommitAuthorAvatarUrl ? (
              <img
                src={deployment.gitCommitAuthorAvatarUrl}
                alt={deployment.gitCommitAuthorHandle}
                className="w-4 h-4 rounded-full shrink-0"
              />
            ) : (
              <User iconSize="sm-regular" className="text-grayA-10 shrink-0" />
            )}
            <span className="truncate max-w-[200px]">{deployment.gitCommitAuthorHandle}</span>
          </div>
        </div>
      )}

      {deployment.gitCommitMessage && (
        <div className="flex items-center justify-between">
          <span className="text-gray-11">Message:</span>
          <span
            className="text-grayA-11 truncate max-w-[250px]"
            title={deployment.gitCommitMessage}
          >
            {truncatedMessage}
          </span>
        </div>
      )}

      {deployment.status && (
        <div className="flex items-center justify-between">
          <span className="text-gray-11">Status:</span>
          <Badge
            variant={
              deployment.status === "ready"
                ? "success"
                : deployment.status === "failed"
                  ? "error"
                  : deployment.status === "building" || deployment.status === "deploying"
                    ? "warning"
                    : "secondary"
            }
            className="text-xs"
          >
            {deployment.status}
          </Badge>
        </div>
      )}
    </div>
  );
};

// Format meta information
const formatMetaInfo = (log: SentinelLogsResponse): React.ReactNode => {
  const timestamp = new Date(log.time);
  const formattedTime = timestamp.toLocaleString("en-US", {
    month: "short",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
    hour12: false,
  });

  // Format query params if present
  const queryParamsEntries = Object.entries(log.query_params);
  const hasQueryParams = queryParamsEntries.length > 0;

  return (
    <div className="flex flex-col gap-2">
      <div className="flex items-center justify-between">
        <span className="text-gray-11">Request ID:</span>
        <div className="flex items-center gap-2">
          <span className="font-mono">{shortenId(log.request_id)}</span>
          <CopyButton value={log.request_id} variant="ghost" className="h-4 w-4" />
        </div>
      </div>
      <div className="flex items-center justify-between">
        <span className="text-gray-11">Timestamp:</span>
        <span className="font-mono">{formattedTime}</span>
      </div>
      <div className="flex items-center justify-between">
        <span className="text-gray-11">IP Address:</span>
        <div className="flex items-center gap-2">
          <span className="font-mono">{log.ip_address}</span>
          <CopyButton value={log.ip_address} variant="ghost" className="h-4 w-4" />
        </div>
      </div>
      <div className="flex items-center justify-between">
        <span className="text-gray-11">User Agent:</span>
        <span className="font-mono truncate max-w-[250px]" title={log.user_agent}>
          {log.user_agent}
        </span>
      </div>
      <div className="flex items-center justify-between">
        <span className="text-gray-11">Region:</span>
        <span className="font-mono">{log.region}</span>
      </div>
      <div className="flex items-center justify-between">
        <span className="text-gray-11">Host:</span>
        <span className="font-mono truncate max-w-[250px]" title={log.host}>
          {log.host}
        </span>
      </div>
      {log.query_string && (
        <div className="flex items-center justify-between">
          <span className="text-gray-11">Query String:</span>
          <span className="font-mono truncate max-w-[250px]" title={log.query_string}>
            {log.query_string}
          </span>
        </div>
      )}
      {hasQueryParams && (
        <div className="flex flex-col gap-1 mt-1">
          <span className="text-gray-11 text-xs">Query Parameters:</span>
          {queryParamsEntries.map(([key, values]) => (
            <div key={key} className="flex items-center justify-between ml-2">
              <span className="text-gray-11 text-xs">{key}:</span>
              <span
                className="font-mono text-xs truncate max-w-[200px]"
                title={(values as string[]).join(", ")}
              >
                {(values as string[]).join(", ")}
              </span>
            </div>
          ))}
        </div>
      )}
    </div>
  );
};
