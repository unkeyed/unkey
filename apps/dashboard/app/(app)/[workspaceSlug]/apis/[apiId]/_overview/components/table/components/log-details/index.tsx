"use client";
import { LogDetails } from "@/components/logs/details/log-details";
import type { KeysOverviewLog } from "@unkey/clickhouse/src/keys/keys";
import { TimestampInfo, toast } from "@unkey/ui";
import Link from "next/link";
import { useEffect, useState } from "react";
import { LogHeader } from "./components/log-header";
import { OutcomeDistributionSection } from "./components/log-outcome-distribution-section";
import { LogSection } from "./components/log-section";
import { PermissionsSection, RolesSection } from "./components/roles-permissions";

const ANIMATION_DELAY = 350;

type Props = {
  distanceToTop: number;
  log: KeysOverviewLog | null;
  apiId: string;
  setSelectedLog: (data: KeysOverviewLog | null) => void;
};

export const KeysOverviewLogDetails = ({ distanceToTop, log, setSelectedLog, apiId }: Props) => {
  const [errorShown, setErrorShown] = useState(false);

  useEffect(() => {
    if (!errorShown && log) {
      if (!log.key_details) {
        toast.error("Key Details Unavailable", {
          description:
            "Could not retrieve key information for this log. The key may have been deleted or is still processing.",
        });
        setErrorShown(true);
      }
    }
    if (!log) {
      setErrorShown(false);
    }
  }, [log, errorShown]);

  const handleClose = () => {
    setSelectedLog(null);
  };

  if (!log) {
    return null;
  }

  if (!log.key_details) {
    return null;
  }

  const metaData = formatMeta(log.key_details.meta);

  const usage = {
    Created: metaData?.createdAt || "N/A",
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
  };

  const identifiers = {
    "Key ID": (
      <Link
        title={`View details for ${log.key_id}`}
        className="font-mono underline decoration-dotted"
        href={`/apis/${apiId}/keys/${log.key_details.key_auth_id}/${log.key_id}`}
      >
        <div className="font-mono font-medium truncate">{log.key_id}</div>
      </Link>
    ),
    Name: log.key_details.name || "N/A",
  };

  const identity = log.key_details.identity
    ? { "External ID": log.key_details.identity.external_id || "N/A" }
    : { "No identity connected": null };

  const tags =
    log.tags && log.tags.length > 0 ? { Tags: log.tags.join(", ") } : { "No tags": null };

  const sections = [
    <LogSection key="usage" title="Usage" details={usage} />,
    log.outcome_counts && (
      <OutcomeDistributionSection key="outcomes" outcomeCounts={log.outcome_counts} />
    ),
    <LogSection key="limits" title="Limits" details={limits} />,
    <LogSection key="identifiers" title="Identifiers" details={identifiers} />,
    <LogSection key="identity" title="Identity" details={identity} />,
    <LogSection key="tags" title="Tags" details={tags} />,
    <RolesSection key="roles" roles={log.key_details.roles || []} />,
    <PermissionsSection key="permissions" permissions={log.key_details.permissions || []} />,
  ].filter(Boolean);

  return (
    <LogDetails
      distanceToTop={distanceToTop}
      log={log || undefined}
      onClose={handleClose}
      isLoading={false}
      error={false}
    >
      <LogDetails.Header onClose={handleClose}>
        <LogHeader log={log} onClose={handleClose} />
      </LogDetails.Header>
      <LogDetails.CustomSections startDelay={150} staggerDelay={50}>
        {sections}
      </LogDetails.CustomSections>
      <LogDetails.Spacer delay={ANIMATION_DELAY} />
      <LogDetails.Meta />
    </LogDetails>
  );
};

// biome-ignore lint/suspicious/noExplicitAny: JSON metadata has unknown structure, runtime validation handles safety
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
