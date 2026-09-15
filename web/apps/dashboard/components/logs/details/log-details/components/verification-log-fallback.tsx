"use client";

import { RequestResponseDetails } from "@/components/logs/details/request-response-details";
import { cn } from "@/lib/utils";
import { XMark } from "@unkey/icons";
import { Badge, Button, TimestampInfo } from "@unkey/ui";
import type { ReactNode } from "react";
import { YELLOW_STATES } from "./log-footer";

type VerificationLog = {
  request_id: string;
  time: number;
  region: string;
  outcome: string;
  tags: string[];
};

type ExtraField = {
  label: string;
  content: string;
};

type Props = {
  log: VerificationLog;
  extraFields?: ExtraField[];
  isLoading: boolean;
  onClose: () => void;
};

const RED_OUTCOMES = ["DISABLED", "FORBIDDEN", "INSUFFICIENT_PERMISSIONS"];

export const VerificationLogFallback = ({ log, extraFields, isLoading, onClose }: Props) => {
  return (
    <>
      <div className="border-b flex justify-between items-center border-gray-4 h-[50px] px-4 py-2">
        <div className="flex gap-2 items-center min-w-0">
          <Badge
            className={cn("uppercase px-[6px] rounded-md font-mono text-xs", {
              "bg-success-3 text-success-11 hover:bg-success-4": log.outcome === "VALID",
              "bg-warning-3 text-warning-11 hover:bg-warning-4": YELLOW_STATES.includes(
                log.outcome,
              ),
              "bg-error-3 text-error-11 hover:bg-error-4": RED_OUTCOMES.includes(log.outcome),
            })}
          >
            {log.outcome || "UNKNOWN"}
          </Badge>
          <p className="text-xs text-accent-12 truncate flex-1 font-mono">{log.request_id}</p>
        </div>
        <Button
          size="icon"
          variant="ghost"
          onClick={onClose}
          className="[&_svg]:size-3"
          aria-label="Close"
        >
          <XMark className="text-grayA-9 stroke-2" iconSize="sm-regular" />
        </Button>
      </div>
      <p className="px-4 pt-4 text-xs text-accent-11">{availabilityCopy(isLoading)}</p>
      <div className="px-4">
        <RequestResponseDetails
          fields={[
            {
              label: "Time",
              description: (content: number) => (
                <TimestampInfo value={content} className="underline decoration-dotted" />
              ),
              content: log.time,
              tooltipContent: "Copy Time",
              tooltipSuccessMessage: "Time copied to clipboard",
              skipTooltip: true,
            },
            ...(extraFields ?? []).map((field) => ({
              label: field.label,
              description: (content: string) => <span className="text-xs font-mono">{content}</span>,
              content: field.content,
              tooltipContent: `Copy ${field.label}`,
              tooltipSuccessMessage: `${field.label} copied to clipboard`,
            })),
            {
              label: "Request ID",
              description: (content: string) => <span className="text-xs font-mono">{content}</span>,
              content: log.request_id,
              tooltipContent: "Copy Request ID",
              tooltipSuccessMessage: "Request ID copied to clipboard",
            },
            {
              label: "Outcome",
              description: (content: string) => (
                <span className="text-xs font-mono uppercase">{content}</span>
              ),
              content: log.outcome || "UNKNOWN",
              tooltipContent: "Copy Outcome",
              tooltipSuccessMessage: "Outcome copied to clipboard",
            },
            {
              label: "Region",
              description: (content: string) => <span className="text-xs font-mono">{content}</span>,
              content: log.region,
              tooltipContent: "Copy Region",
              tooltipSuccessMessage: "Region copied to clipboard",
            },
            {
              label: "Tags",
              description: (content: string) => <span className="text-xs font-mono">{content}</span>,
              content: log.tags.length > 0 ? log.tags.join(", ") : null,
              tooltipContent: "Copy Tags",
              tooltipSuccessMessage: "Tags copied to clipboard",
            },
          ]}
        />
      </div>
    </>
  );
};

function availabilityCopy(isLoading: boolean): ReactNode {
  if (isLoading) {
    return "Loading request headers and body.";
  }

  return "No request headers or body were logged for this verification.";
}
