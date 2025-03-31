"use client";

import { RequestResponseDetails } from "@/components/logs/details/request-response-details";
import { TimestampInfo } from "@/components/timestamp-info";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import type { AuditLog } from "@/lib/trpc/routers/audit/schema";
import { FunctionSquare, KeySquare } from "lucide-react";

type Props = {
  log: AuditLog;
};

export const LogFooter = ({ log }: Props) => {
  return (
    <RequestResponseDetails
      fields={[
        {
          label: "Time",
          description: (content) => (
            <TimestampInfo value={content} className="text-[13px] underline decoration-dotted" />
          ),
          content: log.auditLog.time,
          tooltipContent: "Copy Time",
          tooltipSuccessMessage: "Time copied to clipboard",
          skipTooltip: true,
        },
        {
          label: "Location",
          description: (content) => <span className="text-[13px] font-mono">{content}</span>,
          content: log.auditLog.location,
          tooltipContent: "Copy Location",
          tooltipSuccessMessage: "Location copied to clipboard",
        },
        {
          label: "Actor Details",
          description: (content) => {
            const { log, user } = content;
            return log.actor.type === "user" && user?.imageUrl ? (
              <div className="flex justify-end items-center w-full gap-2 max-sm:m-0 max-sm:gap-1 max-sm:text-xs md:flex-grow">
                <Avatar className="w-6 h-6">
                  <AvatarImage src={user.imageUrl} />
                  <AvatarFallback>{user?.username?.slice(0, 2)}</AvatarFallback>
                </Avatar>
                <span className="text-sm text-content whitespace-nowrap">
                  {`${user?.firstName ?? ""} ${user?.lastName ?? ""}`}
                </span>
              </div>
            ) : log.actor.type === "key" ? (
              <div className="flex items-center w-full gap-2 max-sm:m-0 max-sm:gap-1 max-sm:text-xs md:flex-grow">
                <KeySquare className="w-4 h-4" />
                <span className="font-mono text-xs text-content">{log.actor.id}</span>
              </div>
            ) : (
              <div className="flex items-center w-full gap-2 max-sm:m-0 max-sm:gap-1 max-sm:text-xs md:flex-grow">
                <FunctionSquare className="w-4 h-4" />
                <span className="font-mono text-xs text-content">{log.actor.id}</span>
              </div>
            );
          },
          content: { log: log.auditLog, user: log.user },
          className: "whitespace-pre",
          tooltipContent: "Copy Actor",
          tooltipSuccessMessage: "Actor copied to clipboard",
        },
        {
          label: "User Agent",
          description: (content) => (
            <span className="text-[13px] font-mono text-right">{content}</span>
          ),
          content: log.auditLog.userAgent,
          tooltipContent: "Copy User Agent",
          tooltipSuccessMessage: "User Agent copied to clipboard",
        },
        {
          label: "Event",
          description: (content) => <span className="text-[13px] font-mono">{content}</span>,
          content: log.auditLog.event,
          tooltipContent: "Copy Event",
          tooltipSuccessMessage: "Event copied to clipboard",
        },
        {
          label: "Description",
          description: (content) => <span className="text-[13px] font-mono">{content}</span>,
          content: log.auditLog.description,
          tooltipContent: "Copy Description",
          tooltipSuccessMessage: "Description copied to clipboard",
        },
        {
          label: "Workspace Id",
          description: (content) => <span className="text-[13px] font-mono">{content}</span>,
          content: log.auditLog.workspaceId,
          tooltipContent: "Copy Workspace Id",
          tooltipSuccessMessage: "Workspace Id copied to clipboard",
        },
      ]}
    />
  );
};
