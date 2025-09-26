"use client";

import { extractResponseField, getRequestHeader } from "@/app/(app)/[workspaceSlug]/logs/utils";
import { RequestResponseDetails } from "@/components/logs/details/request-response-details";
import { cn } from "@/lib/utils";
import { Badge, TimestampInfo } from "@unkey/ui";
import type { StandardLogTypes } from "..";

type Props = {
  log: StandardLogTypes;
};

export const YELLOW_STATES = ["RATE_LIMITED", "EXPIRED", "USAGE_EXCEEDED"];
export const RED_STATES = ["DISABLED", "FORBIDDEN", "INSUFFICIENT_PERMISSIONS"];
const DEFAULT_OUTCOME = "VALID";

export const LogFooter = ({ log }: Props) => {
  return (
    <RequestResponseDetails
      fields={[
        {
          label: "Time",
          description: (content) => (
            <TimestampInfo value={content} className="underline decoration-dotted" />
          ),
          content: log.time,
          tooltipContent: "Copy Time",
          tooltipSuccessMessage: "Time copied to clipboard",
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
          label: "Request User Agent",
          description: (content) => <span className="text-xs font-mono text-right">{content}</span>,
          content: getRequestHeader(log, "user-agent") ?? "",
          tooltipContent: "Copy Request User Agent",
          tooltipSuccessMessage: "Request user agent copied to clipboard",
        },
        {
          label: "Outcome",
          description: (content) => {
            let contentCopy = content;
            if (contentCopy == null) {
              contentCopy = DEFAULT_OUTCOME;
            }
            return (
              <Badge
                className={cn(
                  {
                    "text-warning-11 bg-warning-3 hover:bg-warning-3 font-medium":
                      YELLOW_STATES.includes(contentCopy),
                    "text-error-11 bg-error-3 hover:bg-error-3 font-medium":
                      RED_STATES.includes(contentCopy),
                  },
                  "uppercase",
                )}
              >
                {content}
              </Badge>
            );
          },
          content: extractResponseField(log, "code"),
          tooltipContent: "Copy Outcome",
          tooltipSuccessMessage: "Outcome copied to clipboard",
        },
        {
          label: "Permissions",
          description: (content) => (
            <div className="flex flex-wrap gap-2 justify-end">
              {content.map((permission, index) => (
                <Badge
                  variant="secondary"
                  // biome-ignore lint/suspicious/noArrayIndexKey: its okay to use it as a key
                  key={index}
                  className="px-2 py-1 text-xs font-mono rounded-md"
                >
                  {permission}
                </Badge>
              ))}
            </div>
          ),
          content: extractResponseField(log, "permissions"),
          tooltipContent: "Copy Permissions",
          tooltipSuccessMessage: "Permissions copied to clipboard",
        },
      ]}
    />
  );
};
