"use client";

import { safeParseJson } from "@/app/(app)/[workspaceSlug]/logs/utils";
import { EMPTY_TEXT, LogDetails } from "@/components/logs/details/log-details";
import { LogHeader } from "@/components/logs/details/log-details/components/log-header";
import { LogSection } from "@/components/logs/details/log-details/components/log-section";
import { RequestResponseDetails } from "@/components/logs/details/request-response-details";
import { formatLatency } from "@/lib/utils/metric-formatters";
import type { RequestLogsResponse } from "@unkey/clickhouse/src/frontline";
import { TimestampInfo } from "@unkey/ui";
import type { ReactNode } from "react";

type Props = {
  distanceToTop: number;
  log: RequestLogsResponse;
  onClose: () => void;
};

/**
 * Detail panel for a request served by the gateway.
 *
 * Gateways always write a base row, but headers, query data, and bodies are
 * only captured when an enabled logging policy opted the request in. Those
 * sections therefore say the data was never recorded rather than showing
 * `<EMPTY>`, which would read as a request that genuinely had no body.
 */
export const GatewayRequestDetails = ({ distanceToTop, log, onClose }: Props) => {
  return (
    <LogDetails distanceToTop={distanceToTop} log={log} onClose={onClose}>
      <LogDetails.Header>
        <LogHeader log={log} onClose={onClose} />
      </LogDetails.Header>

      <LogDetails.Section>
        <LogSection
          title="Request Header"
          details={log.request_headers.length ? log.request_headers : <NotCaptured />}
        />
      </LogDetails.Section>

      <LogDetails.Section>
        <LogSection title="Request Body" details={formatBody(log.request_body)} />
      </LogDetails.Section>

      <LogDetails.Section>
        <LogSection
          title="Response Header"
          details={log.response_headers.length ? log.response_headers : <NotCaptured />}
        />
      </LogDetails.Section>

      <LogDetails.Section>
        <LogSection title="Response Body" details={formatBody(log.response_body)} />
      </LogDetails.Section>

      <LogDetails.Spacer />

      <div className="px-4">
        <RequestResponseDetails
          fields={[
            {
              label: "Time",
              description: (content) => (
                <TimestampInfo value={content} className="underline decoration-dotted" />
              ),
              content: log.time,
              skipTooltip: true,
            },
            {
              label: "Host",
              description: (content) => <span className="text-xs font-mono">{content}</span>,
              content: log.host,
              tooltipContent: "Copy Host",
              tooltipSuccessMessage: "Host copied to clipboard",
            },
            {
              label: "Request Path",
              description: (content) => <span className="text-xs font-mono">{content}</span>,
              content: log.path,
              tooltipContent: "Copy Request Path",
              tooltipSuccessMessage: "Request path copied to clipboard",
            },
            {
              label: "Request ID",
              description: (content) => <span className="text-xs font-mono">{content}</span>,
              content: log.request_id,
              tooltipContent: "Copy Request ID",
              tooltipSuccessMessage: "Request ID copied to clipboard",
            },
            {
              label: "Region",
              description: (content) => <span className="text-xs font-mono">{content}</span>,
              content: log.region,
              tooltipContent: "Copy Region",
              tooltipSuccessMessage: "Region copied to clipboard",
            },
            {
              label: "Deployment",
              description: (content) => <span className="text-xs font-mono">{content}</span>,
              content: log.deployment_id,
              tooltipContent: "Copy Deployment ID",
              tooltipSuccessMessage: "Deployment ID copied to clipboard",
            },
            {
              label: "Request User Agent",
              description: (content) => (
                <span className="text-xs font-mono text-right">{content}</span>
              ),
              content: log.user_agent,
              tooltipContent: "Copy Request User Agent",
              tooltipSuccessMessage: "Request user agent copied to clipboard",
            },
            {
              label: "Total Latency",
              description: (content) => (
                <span className="text-xs font-mono">{formatLatency(content)}</span>
              ),
              content: log.total_latency,
              skipTooltip: true,
            },
            {
              label: "Gateway Latency",
              description: (content) => (
                <span className="text-xs font-mono">{formatLatency(content)}</span>
              ),
              content: log.gateway_latency,
              skipTooltip: true,
            },
          ]}
        />
      </div>
    </LogDetails>
  );
};

const NotCaptured = () => (
  <span className="text-xs text-grayA-10">
    Not recorded. Gateways capture headers and bodies only for requests an enabled logging policy
    opts in.
  </span>
);

const formatBody = (body: string): ReactNode => {
  if (!body) {
    return <NotCaptured />;
  }

  const formatted = JSON.stringify(safeParseJson(body), null, 2);
  return formatted === "null" ? (
    <span className="text-xs text-accent-12 truncate">{EMPTY_TEXT}</span>
  ) : (
    formatted
  );
};
