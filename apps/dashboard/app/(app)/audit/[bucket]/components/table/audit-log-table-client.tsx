"use client";

import { EmptyPlaceholder } from "@/components/dashboard/empty-placeholder";
import { VirtualTable } from "@/components/virtual-table";
import { trpc } from "@/lib/trpc/client";
import type { User } from "@clerk/nextjs/server";
import { cn } from "@unkey/ui/src/lib/utils";
import { useEffect, useState } from "react";
import { useAuditLogParams } from "../../query-state";
import { columns } from "./columns";
import { DEFAULT_FETCH_COUNT } from "./constants";
import { LogDetails } from "./table-details";
import type { Data } from "./types";
import { getEventType } from "./utils";
import { AuditLogWithTargets } from "@/lib/trpc/routers/audit/fetch";

export const AuditLogTableClient = ({
  data: initialData,
  users,
}: {
  data: AuditLogWithTargets[];
  users: Record<string, User>;
}) => {
  const [selectedLog, setSelectedLog] = useState<Data | null>(null);
  const { setCursor, searchParams } = useAuditLogParams();

  // biome-ignore lint/correctness/useExhaustiveDependencies: including setCursor causes infinite loop
  useEffect(() => {
    // Only set the cursor if we have initial data and no cursor in URL params
    if (initialData.length > 0 && !searchParams.cursor) {
      setCursor(initialData[initialData.length - 1].id);
    }
  }, [initialData, searchParams.cursor]);

  const {
    data,
    fetchNextPage,
    hasNextPage,
    isFetchingNextPage,
    isLoading,
    isError,
  } = trpc.audit.fetch.useInfiniteQuery(
    {
      bucket: searchParams.bucket ?? undefined,
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
      //Breaks the paginated data when refreshing because of cursorTime and cursorId
      staleTime: Number.POSITIVE_INFINITY,
      initialData:
        !searchParams.cursor && initialData.length > 0
          ? {
              pages: [
                {
                  items: initialData,
                  nextCursor: initialData[initialData.length - 1].id,
                },
              ],
              pageParams: [undefined],
            }
          : undefined,
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
      fetchNextPage().then((result) => {
        const lastPage = result.data?.pages[result.data.pages.length - 1];
        if (lastPage?.nextCursor) {
          setCursor(lastPage.nextCursor);
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
      <EmptyPlaceholder>
        <div className="w-[400px] mx-auto flex gap-2 items-center flex-col">
          <div className="text-center">
            <div className="font-medium mb-1">Failed to load audit logs</div>
            <div className="text-sm text-muted-foreground">
              There was a problem fetching the audit logs. Please try refreshing
              the page or contact support if the issue persists.
            </div>
          </div>
        </div>
      </EmptyPlaceholder>
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
      loadingRows={DEFAULT_FETCH_COUNT}
    />
  );
};
