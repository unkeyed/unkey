"use client";

import { TimestampInfo } from "@/components/timestamp-info";
import { Card, CardContent } from "@/components/ui/card";
import { type Column, VirtualTable } from "@/components/virtual-table";
import { trpc } from "@/lib/trpc/client";
import type { User } from "@clerk/nextjs/server";
import {
  ClonePlus2,
  CloneXMark2,
  CodeAction,
  FolderFeather,
} from "@unkey/icons";
import { FunctionSquare, KeySquare, ScrollText } from "lucide-react";
import { useState } from "react";
import { useAuditLogParams } from "./query-state";
import { LogDetails } from "./table-details";

const DEFAULT_FETCH_COUNT = 50;
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
    workspaceId: string | null;
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

export const AuditTable = ({
  data: initialData,
  users,
}: {
  data: Data[];
  users: Record<string, User>;
}) => {
  const [selectedLog, setSelectedLog] = useState<Data | null>(null);

  const { searchParams } = useAuditLogParams();

  const {
    data,
    fetchNextPage,
    hasNextPage,
    isFetchingNextPage,
    isLoading,
    isError,
  } = trpc.audit.fetch.useInfiniteQuery(
    {
      bucket: searchParams.bucket,
      events: searchParams.events,
      users: searchParams.users,
      rootKeys: searchParams.rootKeys,
      limit: DEFAULT_FETCH_COUNT,
    },
    {
      getNextPageParam: (lastPage) => lastPage.nextCursor,
      keepPreviousData: false,
    }
  );

  const flattenedData =
    data?.pages.flatMap((page) =>
      page.items.map((l) => {
        const user = users[l.actorId];
        return {
          user: user
            ? {
                username: user.username,
                firstName: user.firstName,
                lastName: user.lastName,
                imageUrl: user.imageUrl,
              }
            : undefined,
          auditLog: {
            id: l.id,
            time: l.time,
            actor: {
              id: l.actorId,
              name: l.actorName,
              type: l.actorType,
            },
            location: l.remoteIp,
            description: l.display,
            userAgent: l.userAgent,
            event: l.event,
            workspaceId: l.workspaceId,
            targets: l.targets.map((t) => ({
              id: t.id,
              type: t.type,
              name: t.name,
              meta: t.meta,
            })),
          },
        };
      })
    ) ?? [];

  const handleLoadMore = () => {
    if (hasNextPage && !isFetchingNextPage) {
      fetchNextPage();
    }
  };

  const getSelectedClassName = (item_: Data, isSelected: boolean) =>
    isSelected ? "bg-background-subtle/90" : "";

  if (isError) {
    return <div>hehe</div>;
  }

  console.log(flattenedData.length);

  return (
    <VirtualTable
      data={flattenedData}
      columns={columns}
      isLoading={isLoading}
      onLoadMore={handleLoadMore}
      selectedItem={selectedLog}
      onRowClick={setSelectedLog}
      selectedClassName={getSelectedClassName}
      keyExtractor={(log) => log.auditLog.id}
      renderDetails={(log, onClose, distanceToTop) => (
        <LogDetails log={log} onClose={onClose} distanceToTop={distanceToTop} />
      )}
      tableHeight="75vh"
      loadingRows={DEFAULT_FETCH_COUNT}
      overscanCount={10}
      emptyState={
        <Card className="w-[400px] bg-background-subtle">
          <CardContent className="flex justify-center gap-2 py-8">
            <ScrollText />
            <div className="text-sm text-[#666666]">No audit logs found</div>
          </CardContent>
        </Card>
      }
    />
  );
};
