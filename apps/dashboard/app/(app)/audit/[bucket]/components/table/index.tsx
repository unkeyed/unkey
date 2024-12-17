"use client";

import { EmptyPlaceholder } from "@/components/dashboard/empty-placeholder";
import { VirtualTable } from "@/components/virtual-table";
import { trpc } from "@/lib/trpc/client";
import type { User } from "@clerk/nextjs/server";
import { CloneXMark2 } from "@unkey/icons";
import { cn } from "@unkey/ui/src/lib/utils";
import { useEffect, useState } from "react";
import type { AuditLogWithTargets } from "../../page";
import { useAuditLogParams } from "../../query-state";
import { columns } from "./columns";
import { DEFAULT_FETCH_COUNT } from "./constants";
import { LogDetails } from "./table-details";
import type { Data } from "./types";
import { getEventType } from "./utils";

export const AuditTable = ({
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
    if (initialData.length > 0 && !searchParams.cursorId) {
      setCursor({
        time: initialData[initialData.length - 1].time,
        id: initialData[initialData.length - 1].id,
      });
    }
  }, [initialData, searchParams.cursorId]);

  const { data, fetchNextPage, hasNextPage, isFetchingNextPage, isLoading, isError } =
    trpc.audit.fetch.useInfiniteQuery(
      {
        bucket: searchParams.bucket,
        limit: DEFAULT_FETCH_COUNT,
        users: searchParams.users,
        events: searchParams.events,
        rootKeys: searchParams.rootKeys,
      },
      {
        initialCursor: searchParams.cursorId
          ? {
              time: searchParams.cursorTime,
              id: searchParams.cursorId,
            }
          : undefined,
        getNextPageParam: (lastPage) => {
          return lastPage.nextCursor;
        },
        //Break the paginated data when refreshing because of cursorTime and cursorId
        staleTime: Number.POSITIVE_INFINITY,
        keepPreviousData: false,
        initialData:
          !searchParams.cursorId && initialData.length > 0
            ? {
                pages: [
                  {
                    items: initialData,
                    nextCursor:
                      initialData.length === DEFAULT_FETCH_COUNT
                        ? {
                            time: initialData[initialData.length - 1].time,
                            id: initialData[initialData.length - 1].id,
                          }
                        : undefined,
                  },
                ],
                pageParams: [undefined],
              }
            : undefined,
      },
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
      }),
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
      "text-error-12 hover:bg-error-3": eventType === "delete",
      "text-warning-12 hover:bg-warning-3": eventType === "update",
      "text-success-12 hover:bg-success-3": eventType === "create",
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
          <CloneXMark2 className="w-8 h-8 text-[hsl(var(--error-11))]" />
          <div className="text-center">
            <div className="font-medium mb-1">Failed to load audit logs</div>
            <div className="text-sm text-muted-foreground">
              There was a problem fetching the audit logs. Please try refreshing the page or contact
              support if the issue persists.
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
      tableHeight="75vh"
      loadingRows={DEFAULT_FETCH_COUNT}
    />
  );
};
