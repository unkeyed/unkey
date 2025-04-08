"use client";
import { DEFAULT_DRAGGABLE_WIDTH } from "@/app/(app)/logs/constants";
import { ResizablePanel } from "@/components/logs/details/resizable-panel";
import { TimestampInfo } from "@/components/timestamp-info";
import type { KeysOverviewLog } from "@unkey/clickhouse/src/keys/keys";
import Link from "next/link";
import { useMemo } from "react";
import { LogHeader } from "./components/log-header";
import { OutcomeDistributionSection } from "./components/log-outcome-distribution-section";
import { LogSection } from "./components/log-section";
import { PermissionsSection, RolesSection } from "./components/roles-permissions";

type StyleObject = {
  top: string;
  width: string;
  height: string;
  paddingBottom: string;
};

const createPanelStyle = (distanceToTop: number): StyleObject => ({
  top: `${distanceToTop}px`,
  width: `${DEFAULT_DRAGGABLE_WIDTH}px`,
  height: `calc(100vh - ${distanceToTop}px)`,
  paddingBottom: "1rem",
});

type KeysOverviewLogDetailsProps = {
  distanceToTop: number;
  log: KeysOverviewLog | null;
  apiId: string;
  setSelectedLog: (data: KeysOverviewLog | null) => void;
};

export const KeysOverviewLogDetails = ({
  distanceToTop,
  log,
  setSelectedLog,
  apiId,
}: KeysOverviewLogDetailsProps) => {
  const panelStyle = useMemo(() => createPanelStyle(distanceToTop), [distanceToTop]);

  if (!log) {
    return null;
  }

  const handleClose = (): void => {
    setSelectedLog(null);
  };

  // Only process if key_details exists
  if (!log.key_details) {
    return (
      <ResizablePanel
        onClose={handleClose}
        className="absolute right-0 bg-gray-1 dark:bg-black font-mono drop-shadow-2xl overflow-y-auto z-20 p-4"
        style={panelStyle}
      >
        <LogHeader log={log} onClose={handleClose} />
        <div className="py-4 text-center text-accent-9">No key details available</div>
      </ResizablePanel>
    );
  }

  // Process key details data
  const metaData = formatMeta(log.key_details.meta);
  const identifiers = {
    "Key ID": (
      <Link
        title={`View details for ${log.key_id}`}
        className="font-mono underline decoration-dotted"
        href={`/apis/${apiId}/keys/${log.key_details?.key_auth_id}/${log.key_id}`}
      >
        <div className="font-mono font-medium truncate">{log.key_id}</div>
      </Link>
    ),
    Name: log.key_details.name || "N/A",
  };

  const usage = {
    Created: metaData?.createdAt ? metaData.createdAt : "N/A",
    "Last Used": log.time ? (
      <TimestampInfo value={log.time} className="font-mono underline decoration-dotted" />
    ) : (
      "N/A"
    ),
  };

  const limits = {
    Status: log.key_details.enabled ? "Enabled" : "Disabled",
    Remaining:
      log.key_details.remaining_requests !== null
        ? log.key_details.remaining_requests
        : "Unlimited",
    "Rate Limit": log.key_details.ratelimit_limit
      ? `${log.key_details.ratelimit_limit} per ${log.key_details.ratelimit_duration || "N/A"}ms`
      : "No limit",
    Async: log.key_details.ratelimit_async ? "Yes" : "No",
  };

  const identity = log.key_details.identity
    ? { "External ID": log.key_details.identity.external_id || "N/A" }
    : { "No identity connected": null };

  const metaString = metaData ? JSON.stringify(metaData, null, 2) : { "No meta available": "" };

  return (
    <ResizablePanel
      onClose={handleClose}
      className="absolute right-0 bg-gray-1 dark:bg-black font-mono shadow-2xl overflow-y-auto z-20 p-4"
      style={panelStyle}
    >
      <LogHeader log={log} onClose={handleClose} />
      <LogSection title="Usage" details={usage} />
      {log.outcome_counts && <OutcomeDistributionSection outcomeCounts={log.outcome_counts} />}
      <LogSection title="Limits" details={limits} />
      <LogSection title="Identifiers" details={identifiers} />
      <LogSection title="Identity" details={identity} />
      <RolesSection roles={log.key_details.roles || []} />
      <PermissionsSection permissions={log.key_details.permissions || []} />
      <LogSection title="Meta" details={metaString} />
    </ResizablePanel>
  );
};

const formatMeta = (meta: string | null): Record<string, any> | null => {
  if (!meta) {
    return null;
  }
  try {
    const parsedMeta = JSON.parse(meta);
    return parsedMeta;
  } catch {
    return null;
  }
};
