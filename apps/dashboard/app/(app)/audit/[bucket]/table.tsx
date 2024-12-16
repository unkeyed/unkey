"use client";

import { TimestampInfo } from "@/components/timestamp-info";
import { Column, VirtualTable } from "@/components/virtual-table";
import {
  ClonePlus2,
  CloneXMark2,
  CodeAction,
  FolderFeather,
} from "@unkey/icons";
import { cn } from "@unkey/ui/src/lib/utils";
import { FunctionSquare, KeySquare } from "lucide-react";
import { useState } from "react";
import { LogDetails } from "./table-details";

export type Data = {
  user:
    | {
        username?: string | null;
        firstName?: string | null;
        lastName?: string | null;
        imageUrl?: string | null;
      }
    | undefined;
  auditLog: {
    id: string;
    time: number;
    actor: {
      id: string;
      type: string;
      name: string | null;
    };
    event: string;
    location: string | null;
    userAgent: string | null;
    workspaceId: string;
    targets: Array<{
      id: string;
      type: string;
      name: string | null;
      meta: unknown;
    }>;
    description: string;
  };
};

const EventTypeIcon = ({ event }: { event: string }) => {
  const eventLower = event.toLowerCase();
  if (eventLower.includes("delete")) {
    return <CloneXMark2 className="text-[hsl(var(--error-11))]" />;
  }
  if (eventLower.includes("update")) {
    return <FolderFeather className="text-[hsl(var(--warning-11))]" />;
  }
  if (eventLower.includes("create")) {
    return <ClonePlus2 className="text-[hsl(var(--success-11))]" />;
  }
  return <CodeAction />;
};

export const columns: Column<Data>[] = [
  {
    key: "time",
    header: "Time",
    width: "166px",
    render: (log) => <TimestampInfo value={log.auditLog.time} />,
  },
  {
    key: "actor",
    header: "Actor",
    width: "10%",
    render: (log) => (
      <div className="flex items-center">
        {log.auditLog.actor.type === "user" && log.user ? (
          <div className="flex items-center w-full gap-2 max-sm:m-0 max-sm:gap-1 max-sm:text-xs">
            <span className="text-xs whitespace-nowrap">{`${
              log.user.firstName ?? ""
            } ${log.user.lastName ?? ""}`}</span>
          </div>
        ) : log.auditLog.actor.type === "key" ? (
          <div className="flex items-center w-full gap-2 max-sm:m-0 max-sm:gap-1 max-sm:text-xs">
            <KeySquare className="w-4 h-4" />
            <span className="font-mono text-xs">{log.auditLog.actor.id}</span>
          </div>
        ) : (
          <div className="flex items-center w-full gap-2 max-sm:m-0 max-sm:gap-1 max-sm:text-xs">
            <FunctionSquare className="w-4 h-4" />
            <span className="font-mono text-xs">{log.auditLog.actor.id}</span>
          </div>
        )}
      </div>
    ),
  },
  {
    key: "event",
    header: "Event",
    width: "20%",
    render: (log) => (
      <div className="flex items-center gap-2 text-current font-mono text-xs">
        <EventTypeIcon event={log.auditLog.event} />
        <span>{log.auditLog.event}</span>
      </div>
    ),
  },
  {
    key: "event-description",
    header: "Description",
    width: "auto",
    render: (log) => (
      <div className="text-current font-mono px-2 text-xs">
        {log.auditLog.description}
      </div>
    ),
  },
];

export const AuditTable = ({ data }: { data: Data[] }) => {
  const [selectedLog, setSelectedLog] = useState<Data | null>(null);

  const getSelectedClassName = (_item: Data, isSelected: boolean) =>
    isSelected ? cn("bg-background-subtle/90") : "";

  return (
    <VirtualTable
      data={data}
      columns={columns}
      selectedClassName={getSelectedClassName}
      onRowClick={setSelectedLog}
      selectedItem={selectedLog}
      tableHeight="75vh"
      keyExtractor={(log) => log.auditLog.id}
      renderDetails={(log, onClose, distanceToTop) => (
        <LogDetails log={log} onClose={onClose} distanceToTop={distanceToTop} />
      )}
    />
  );
};
