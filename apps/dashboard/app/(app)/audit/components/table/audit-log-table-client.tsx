"use client";

import { VirtualTable } from "@/components/virtual-table";
import { trpc } from "@/lib/trpc/client";
import { Empty } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import { useState } from "react";
import { useAuditLogParams } from "../../query-state";
import { columns } from "./columns";
import { DEFAULT_FETCH_COUNT } from "./constants";
import { LogDetails } from "./table-details";
import type { Data } from "./types";
import { getEventType } from "./utils";

export const AuditLogTableClient = () => {
  const [selectedLog, setSelectedLog] = useState<Data | null>(null);
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
        getNextPageParam: (lastPage) => {
          return lastPage.nextCursor;
        },
        initialCursor: searchParams.cursor,
        staleTime: Number.POSITIVE_INFINITY,
        refetchOnMount: false,
        refetchOnWindowFocus: false,
      },
    );

  const flattenedData = data?.pages.flatMap((page) => page.items) ?? [];

  const handleLoadMore = () => {
    if (hasNextPage && !isFetchingNextPage && data?.pages.length) {
      // Get the current last page before fetching next
      const currentLastPage = data.pages[data.pages.length - 1];

      fetchNextPage().then(() => {
        // Set the cursor to the last page we had before fetching
        if (currentLastPage.nextCursor) {
          setCursor(currentLastPage.nextCursor);
        }
      });
    }
  };
  const getRowClassName = (item: Data) => {
    const eventType = getEventType(item.auditLog.event);
    return cn({
      "hover:bg-error-3": eventType === "delete",
      "hover:bg-warning-3": eventType === "update",
      "hover:bg-success-3": eventType === "create",
    });
  };

  const getSelectedClassName = (item: Data, isSelected: boolean) => {
    if (!isSelected) {
      return "";
    }

    const eventType = getEventType(item.auditLog.event);
    return cn({
      "bg-error-3": eventType === "delete",
      "bg-warning-3": eventType === "update",
      "bg-success-3": eventType === "create",
      "bg-accent-3": eventType === "other",
    });
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
      renderDetails={(log, onClose, distanceToTop) => (
        <LogDetails log={log} onClose={onClose} distanceToTop={distanceToTop} />
      )}
      config={{
        loadingRows: DEFAULT_FETCH_COUNT,
      }}
    />
  );
};
