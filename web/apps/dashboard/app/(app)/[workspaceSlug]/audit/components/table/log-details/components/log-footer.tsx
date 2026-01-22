"use client";

import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import type { AuditLog } from "@/lib/trpc/routers/audit/schema";
import { Key, MathFunction } from "@unkey/icons";
import { TimestampInfo } from "@unkey/ui";
import { LogSection } from "./log-section";

type Props = {
  log: AuditLog;
};

export const LogFooter = ({ log }: Props) => {
  const actorContent =
    log.auditLog.actor.type === "user" && log.user?.imageUrl ? (
      <div className="flex items-center gap-2">
        <Avatar className="w-6 h-6">
          <AvatarImage src={log.user.imageUrl} />
          <AvatarFallback>{log.user?.username?.slice(0, 2)}</AvatarFallback>
        </Avatar>
        <span className="text-sm">{log.user.username}</span>
      </div>
    ) : log.auditLog.actor.type === "key" ? (
      <div className="flex items-center gap-2">
        <Key iconSize="sm-thin" />
        <span className="font-mono text-xs">{log.auditLog.actor.id}</span>
      </div>
    ) : (
      <div className="flex items-center gap-2">
        <MathFunction iconSize="sm-thin" />
        <span className="font-mono text-xs">{log.auditLog.actor.id}</span>
      </div>
    );

  const overview = {
    Time: <TimestampInfo value={log.auditLog.time} className="underline decoration-dotted" />,
    Event: log.auditLog.event,
    Description: log.auditLog.description,
  };

  const actor = {
    Type: log.auditLog.actor.type.charAt(0).toUpperCase() + log.auditLog.actor.type.slice(1),
    Details: actorContent,
  };

  const details = {
    Location: log.auditLog.location || "N/A",
    "User Agent": log.auditLog.userAgent || "N/A",
    "Workspace ID": log.auditLog.workspaceId,
  };

  return (
    <>
      <LogSection title="Overview" details={overview} />
      <LogSection title="Actor" details={actor} />
      <LogSection title="Details" details={details} />
    </>
  );
};
