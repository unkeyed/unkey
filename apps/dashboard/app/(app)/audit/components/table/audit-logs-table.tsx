"use client";

import { VirtualTable } from "@/components/virtual-table";
import { trpc } from "@/lib/trpc/client";
import { Empty } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import type { AuditData } from "../../audit.type";
import { useAuditLogParams } from "../../query-state";
import { columns } from "./columns";
import { DEFAULT_FETCH_COUNT } from "./constants";
import { getEventType } from "./utils";

const STATUS_STYLES: Record<
  "create" | "update" | "delete" | "other",
  { base: string; hover: string; selected: string }
> = {
  create: {
    base: "text-accent-11",
    hover: "hover:bg-accent-3",
    selected: "bg-accent-3",
  },
  other: {
    base: "text-accent-11 ",
    hover: "hover:bg-accent-3",
    selected: "bg-accent-3",
  },
  update: {
    base: "text-warning-11 ",
    hover: "hover:bg-warning-3",
    selected: "bg-warning-3",
  },
  delete: {
    base: "text-error-11",
    hover: "hover:bg-error-3",
    selected: "bg-error-3",
  },
};

type Props = {
  selectedLog: AuditData | null;
  setSelectedLog: (log: AuditData | null) => void;
};

export const AuditLogsTable = ({ selectedLog, setSelectedLog }: Props) => {
  const { setCursor, searchParams } = useAuditLogParams();

  const { data, fetchNextPage, hasNextPage, isFetchingNextPage, isLoading, isError } =
    trpc.audit.fetch.useInfiniteQuery(
      {
        bucketName: searchParams.bucket ?? undefined,
        limit: DEFAULT_FETCH_COUNT,
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
      },
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

  const getRowClassName = (item: AuditData) => {
    const eventType = getEventType(item.auditLog.event);
    const style = STATUS_STYLES[eventType];

    return cn(
      style.base,
      style.hover,
      "group rounded-md",
      selectedLog && {
        "opacity-50 z-0": selectedLog.auditLog.id !== item.auditLog.id,
        "opacity-100 z-10": selectedLog.auditLog.id === item.auditLog.id,
      },
    );
  };

  const getSelectedClassName = (item: AuditData, isSelected: boolean) => {
    if (!isSelected) {
      return "";
    }
    const style = STATUS_STYLES[getEventType(item.auditLog.event)];
    return style.selected;
  };

  if (isError) {
    return (
      <Empty>
        <Empty.Title>Failed to load audit logs</Empty.Title>
        <Empty.Description>
          There was a problem fetching the audit logs. Please try refreshing the page or contact
          support if the issue persists.
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
      rowClassName={getRowClassName}
      selectedItem={selectedLog}
      onRowClick={setSelectedLog}
      selectedClassName={getSelectedClassName}
      keyExtractor={(log) => log.auditLog.id}
      config={{
        loadingRows: DEFAULT_FETCH_COUNT,
      }}
    />
  );
};
