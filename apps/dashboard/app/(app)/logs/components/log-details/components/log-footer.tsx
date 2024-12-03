"use client";
import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";
import { format } from "date-fns";
import { RED_STATES, YELLOW_STATES } from "../../../constants";
import type { Log } from "../../../types";
import { getRequestHeader, getResponseBodyFieldOutcome } from "../../../utils";
import { MetaContent } from "./meta-content";
import { RequestResponseDetails } from "./request-response-details";

type Props = {
  log: Log;
};
const DEFAULT_OUTCOME = "VALID";
export const LogFooter = ({ log }: Props) => {
  return (
    <RequestResponseDetails
      className="pl-3"
      fields={[
        {
          label: "Time",
          description: (content) => (
            <span className="text-[13px] font-mono">{content}</span>
          ),
          content: format(log.time, "MMM dd HH:mm:ss.SS"),
          tooltipContent: "Copy Time",
          tooltipSuccessMessage: "Time copied to clipboard",
        },
        {
          label: "Host",
          description: (content) => (
            <span className="text-[13px] font-mono">{content}</span>
          ),
          content: log.host,
          tooltipContent: "Copy Host",
          tooltipSuccessMessage: "Host copied to clipboard",
        },
        {
          label: "Request Path",
          description: (content) => (
            <span className="text-[13px] font-mono">{content}</span>
          ),
          content: log.path,
          tooltipContent: "Copy Request Path",
          tooltipSuccessMessage: "Request path copied to clipboard",
        },
        {
          label: "Request ID",
          description: (content) => (
            <span className="text-[13px] font-mono">{content}</span>
          ),
          content: log.request_id,
          tooltipContent: "Copy Request ID",
          tooltipSuccessMessage: "Request ID copied to clipboard",
        },
        {
          label: "Request User Agent",
          description: (content) => (
            <span className="text-[13px] font-mono">{content}</span>
          ),
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
                    "text-amber-11 bg-amber-3 hover:bg-amber-3 font-medium":
                      YELLOW_STATES.includes(contentCopy),
                    "text-red-11 bg-red-3 hover:bg-red-3 font-medium":
                      RED_STATES.includes(contentCopy),
                  },
                  "uppercase"
                )}
              >
                {content}
              </Badge>
            );
          },
          content: getResponseBodyFieldOutcome(log, "code"),
          tooltipContent: "Copy Outcome",
          tooltipSuccessMessage: "Outcome copied to clipboard",
        },
        {
          label: "Permissions",
          description: (content) => (
            <span className="text-[13px] font-mono flex gap-1">
              {content.map((permission) => (
                <Badge key={permission} variant="secondary">
                  {permission}
                </Badge>
              ))}
            </span>
          ),
          content: getResponseBodyFieldOutcome(log, "permissions"),
          tooltipContent: "Copy Permissions",
          tooltipSuccessMessage: "Permissions copied to clipboard",
        },
        {
          label: "Meta",
          description: (content) => <MetaContent content={content} />,
          content: getResponseBodyFieldOutcome(log, "meta"),
          tooltipContent: "Copy Meta",
          tooltipSuccessMessage: "Meta copied to clipboard",
        },
      ]}
    />
  );
};
