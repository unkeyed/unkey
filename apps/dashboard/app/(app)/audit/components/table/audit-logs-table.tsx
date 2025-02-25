"use client";
import { TimestampInfo } from "@/components/timestamp-info";
import { VirtualTable } from "@/components/virtual-table";
import type { Column } from "@/components/virtual-table/types";
import { trpc } from "@/lib/trpc/client";
import { Empty } from "@unkey/ui";
import type { AuditData } from "../../audit.type";
import {
  getAuditRowClassName,
  getAuditSelectedClassName,
  getAuditStatusStyle,
  getEventType,
} from "./utils/get-row-class";
import { FunctionSquare, KeySquare } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { cn } from "@unkey/ui/src/lib/utils";
import { useAuditLogParams } from "../../query-state";

type Props = {
  selectedLog: AuditData | null;
  setSelectedLog: (log: AuditData | null) => void;
};

export const AuditLogsTable = ({ selectedLog, setSelectedLog }: Props) => {
  const { setCursor, searchParams } = useAuditLogParams();

  const {
    data,
    fetchNextPage,
    hasNextPage,
    isFetchingNextPage,
    isLoading,
    isError,
  } = trpc.audit.fetch.useInfiniteQuery(
    {
      bucketName: searchParams.bucket ?? undefined,
      limit: 50,
      users: searchParams.users,
      events: searchParams.events,
      rootKeys: searchParams.rootKeys,
      startTime: searchParams.startTime,
      endTime: searchParams.endTime,
    },
    {
      getNextPageParam: (lastPage) => lastPage.nextCursor,
      initialCursor: searchParams.cursor,
      staleTime: Number.POSITIVE_INFINITY,
      refetchOnMount: false,
      refetchOnWindowFocus: false,
    }
  );

  const flattenedData = data?.pages.flatMap((page) => page.items) ?? [];

  const handleLoadMore = () => {
    if (hasNextPage && !isFetchingNextPage && data?.pages.length) {
      const currentLastPage = data.pages[data.pages.length - 1];
      fetchNextPage().then(() => {
        if (currentLastPage.nextCursor) {
          setCursor(currentLastPage.nextCursor);
        }
      });
    }
  };

  if (isError) {
    return (
      <Empty>
        <Empty.Title>Failed to load audit logs</Empty.Title>
        <Empty.Description>
          There was a problem fetching the audit logs. Please try refreshing the
          page or contact support if the issue persists.
        </Empty.Description>
      </Empty>
    );
  }

  return (
    <VirtualTable
      data={flattenedData}
      columns={columns}
      isLoading={isLoading}
      isFetchingNextPage={isFetchingNextPage}
      onLoadMore={handleLoadMore}
      rowClassName={(log) =>
        getAuditRowClassName(
          log,
          selectedLog?.auditLog.id === log.auditLog.id,
          Boolean(selectedLog)
        )
      }
      selectedItem={selectedLog}
      onRowClick={setSelectedLog}
      selectedClassName={getAuditSelectedClassName}
      keyExtractor={(log) => log.auditLog.id}
      config={{
        loadingRows: 50,
      }}
    />
  );
};

export const columns: Column<AuditData>[] = [
  {
    key: "time",
    header: "Time",
    width: "150px",
    headerClassName: "pl-2",
    render: (log) => {
      return (
        <TimestampInfo
          value={log.auditLog.time}
          className="font-mono group-hover:underline decoration-dotted pl-2"
        />
      );
    },
  },
  {
    key: "actor",
    header: "Actor",
    width: "15%",
    render: (log) => (
      <div className="flex items-center gap-3 truncate">
        {log.auditLog.actor.type === "user" && log.user ? (
          <div className="flex items-center w-full gap-2 max-sm:m-0 max-sm:gap-1 max-sm:text-xs">
            <span className="text-xs whitespace-nowrap">
              {`${log.user.firstName ?? ""} ${log.user.lastName ?? ""}`}
            </span>
          </div>
        ) : log.auditLog.actor.type === "key" ? (
          <div className="flex items-center w-full gap-2 max-sm:m-0 max-sm:gap-1 max-sm:text-xs">
            <KeySquare className="w-4 h-4" />
            <span className="font-mono text-xs truncate">
              {log.auditLog.actor.id}
            </span>
          </div>
        ) : (
          <div className="flex items-center w-full gap-2 max-sm:m-0 max-sm:gap-1 max-sm:text-xs">
            <FunctionSquare className="w-4 h-4" />
            <span className="font-mono text-xs truncate">
              {log.auditLog.actor.id}
            </span>
          </div>
        )}
      </div>
    ),
  },
  {
    key: "action",
    header: "Action",
    width: "15%",
    render: (log) => {
      const eventType = getEventType(log.auditLog.event);
      const style = getAuditStatusStyle(log);

      return (
        <div className="flex items-center gap-3 group/action">
          <Badge
            className={cn(
              "uppercase px-[6px] rounded-md font-mono whitespace-nowrap",
              style.badge.default
            )}
          >
            {eventType}
          </Badge>
        </div>
      );
    },
  },
  {
    key: "event",
    header: "Event",
    width: "20%",
    render: (log) => (
      <div className="flex items-center gap-2 font-mono text-xs truncate">
        <span>{log.auditLog.event}</span>
      </div>
    ),
  },
  {
    key: "event-description",
    header: "Description",
    width: "auto",
    render: (log) => (
      <div className="font-mono text-xs truncate w-[200px]">
        {log.auditLog.description}
      </div>
    ),
  },
];
