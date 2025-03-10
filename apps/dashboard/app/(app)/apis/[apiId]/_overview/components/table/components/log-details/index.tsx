"use client";
import { DEFAULT_DRAGGABLE_WIDTH } from "@/app/(app)/logs/constants";
import { ResizablePanel } from "@/components/logs/details/resizable-panel";
import type { KeysOverviewLog } from "@unkey/clickhouse/src/keys/keys";
import { useMemo } from "react";
import { LogHeader } from "./components/log-header";
import { OutcomeDistributionSection } from "./components/log-outcome-distribution-section";
import { LogSection } from "./components/log-section";
import { SummarySection } from "./components/summary-section";

const createPanelStyle = (distanceToTop: number) => ({
  top: `${distanceToTop}px`,
  width: `${DEFAULT_DRAGGABLE_WIDTH}px`,
  height: `calc(100vh - ${distanceToTop}px)`,
  paddingBottom: "1rem",
});

type Props = {
  distanceToTop: number;
  log: KeysOverviewLog | null;
  setSelectedLog: (data: KeysOverviewLog | null) => void;
};

export const KeysOverviewLogDetails = ({ distanceToTop, log, setSelectedLog }: Props) => {
  const panelStyle = useMemo(() => createPanelStyle(distanceToTop), [distanceToTop]);

  if (!log) {
    return null;
  }

  const handleClose = () => {
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
  const createdAt = metaData?.createdAt
    ? formatDate(metaData.createdAt.replace(/3NZ$/, "3Z"))
    : "N/A";

  const summaryStats = [
    `Valid Requests: ${log.valid_count || 0}`,
    `Error Requests: ${log.error_count || 0}`,
    `Age: ${calculateAge(metaData)}`,
  ];

  const identifiers = [
    `ID: ${log.key_details.id}`,
    `Auth ID: ${log.key_details.key_auth_id}`,
    `Name: ${log.key_details.name || "N/A"}`,
    `Workspace: ${log.key_details.workspace_id}`,
  ];

  const usage = [
    `Created: ${createdAt}`,
    `Last Used: ${log.time ? formatDate(log.time) : "N/A"}`,
    `Unique ID: ${metaData?.uniqueId || "N/A"}`,
  ];

  const limits = [
    `Status: ${log.key_details.enabled ? "Enabled" : "Disabled"}`,
    `Remaining: ${
      log.key_details.remaining_requests !== null ? log.key_details.remaining_requests : "Unlimited"
    }`,
    `Rate Limit: ${
      log.key_details.ratelimit_limit
        ? `${log.key_details.ratelimit_limit} per ${log.key_details.ratelimit_duration || "N/A"}s`
        : "No limit"
    }`,
    `Async: ${log.key_details.ratelimit_async ? "Yes" : "No"}`,
  ];

  const refills = [
    `Refill Amount: ${
      log.key_details.refill_amount !== null ? log.key_details.refill_amount : "N/A"
    }`,
    `Refill Day: ${log.key_details.refill_day !== null ? log.key_details.refill_day : "N/A"}`,
    `Last Refill: ${formatDate(log.key_details.last_refill_at)}`,
    `Expires: ${formatDate(log.key_details.expires)}`,
  ];

  const metaString = metaData ? JSON.stringify(metaData, null, 2) : "<EMPTY>";

  return (
    <ResizablePanel
      onClose={handleClose}
      className="absolute right-0 bg-gray-1 dark:bg-black font-mono shadow-2xl overflow-y-auto z-20 p-4"
      style={panelStyle}
    >
      <LogHeader log={log} onClose={handleClose} />

      <SummarySection summaryStats={summaryStats} />
      <LogSection title="Identifiers" details={identifiers} />
      <LogSection title="Usage" details={usage} />
      <LogSection title="Limits" details={limits} />
      <LogSection title="Refills & Expiration" details={refills} />

      {log.outcome_counts && <OutcomeDistributionSection outcomeCounts={log.outcome_counts} />}

      <LogSection title="Meta" details={metaString} />
    </ResizablePanel>
  );
};

const formatDate = (date: string | number | Date | null) => {
  if (!date) {
    return "N/A";
  }
  try {
    if (date instanceof Date) {
      return date.toLocaleString();
    }
    return new Date(date).toLocaleString();
  } catch {
    return "Invalid Date";
  }
};

const formatMeta = (meta: string | null) => {
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

const calculateAge = (metaData: Record<string, any>) => {
  if (!metaData?.createdAt) {
    return "N/A";
  }
  try {
    const created = new Date(metaData.createdAt.replace(/3NZ$/, "3Z"));
    const now = new Date();
    const diffInDays = Math.floor((now.getTime() - created.getTime()) / (1000 * 60 * 60 * 24));

    if (diffInDays === 0) {
      const diffInHours = Math.floor((now.getTime() - created.getTime()) / (1000 * 60 * 60));
      return diffInHours === 0
        ? "Less than an hour"
        : `${diffInHours} hour${diffInHours === 1 ? "" : "s"}`;
    }

    return `${diffInDays} day${diffInDays === 1 ? "" : "s"}`;
  } catch {
    return "N/A";
  }
};
