"use client";

import { VirtualTable } from "@/components/virtual-table/index";
import type { Column } from "@/components/virtual-table/types";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { shortenId } from "@/lib/shorten-id";
import { trpc } from "@/lib/trpc/client";
import type { IdentityResponseSchema } from "@/lib/trpc/routers/identity/query";
import { BookBookmark, Dots, Fingerprint } from "@unkey/icons";
import { Button, CopyButton, Empty, InfoTooltip, Loading } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import dynamic from "next/dynamic";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { parseAsString, useQueryState } from "nuqs";
import { useCallback, useEffect, useMemo, useState } from "react";
import type { z } from "zod";
import { LastUsedCell } from "./last-used";
import {
  ActionColumnSkeleton,
  CountColumnSkeleton,
  CreatedColumnSkeleton,
  IdentityColumnSkeleton,
  LastUsedColumnSkeleton,
} from "./skeletons";

type Identity = z.infer<typeof IdentityResponseSchema>;

const IdentityTableActionPopover = dynamic(
  () => import("./identity-table-actions").then((mod) => mod.IdentityTableActions),
  {
    ssr: false,
    loading: () => (
      <button
        type="button"
        className={cn(
          "group-data-[state=open]:bg-gray-6 group-hover:bg-gray-6 group size-5 p-0 rounded m-0 items-center flex justify-center",
          "border border-gray-6 group-hover:border-gray-8 ring-2 ring-transparent focus-visible:ring-gray-7 focus-visible:border-gray-7",
        )}
      >
        <Dots className="group-hover:text-gray-12 text-gray-11" iconSize="sm-regular" />
      </button>
    ),
  },
);

export const IdentitiesList = () => {
  const [search] = useQueryState(
    "search",
    parseAsString.withDefault("").withOptions({
      history: "replace",
      shallow: true,
      clearOnDefault: true,
    }),
  );
  const workspace = useWorkspaceNavigation();
  const router = useRouter();
  const [identitiesMap, setIdentitiesMap] = useState(() => new Map<string, Identity>());
  const [selectedIdentity, setSelectedIdentity] = useState<Identity | null>(null);
  const [navigatingIdentityId, setNavigatingIdentityId] = useState<string | null>(null);
  const [totalCount, setTotalCount] = useState(0);

  const identitiesList = useMemo(() => Array.from(identitiesMap.values()), [identitiesMap]);

  const {
    data: identitiesData,
    hasNextPage,
    fetchNextPage,
    isFetchingNextPage,
    isLoading: isLoadingInitial,
  } = trpc.identity.query.useInfiniteQuery(
    {
      limit: 50,
      search,
    },
    {
      getNextPageParam: (lastPage) => lastPage.nextCursor,
      staleTime: Number.POSITIVE_INFINITY,
      refetchOnMount: false,
      refetchOnWindowFocus: false,
    },
  );

  useEffect(() => {
    if (identitiesData) {
      const newMap = new Map<string, Identity>();
      identitiesData.pages.forEach((page) => {
        page.identities.forEach((identity) => {
          newMap.set(identity.id, identity);
        });
      });
      if (identitiesData.pages.length > 0) {
        setTotalCount(identitiesData.pages[0].totalCount);
      }
      setIdentitiesMap(newMap);
    }
  }, [identitiesData]);

  const handleLinkClick = useCallback((identityId: string) => {
    setNavigatingIdentityId(identityId);
    setSelectedIdentity(null);
  }, []);

  const handleRowClick = useCallback(
    (identity: Identity) => {
      router.push(`/${workspace.slug}/identities/${identity.id}`);
    },
    [router, workspace.slug],
  );

  const columns: Column<Identity>[] = useMemo(
    () => [
      {
        key: "externalId",
        header: "External ID",
        width: "20%",
        headerClassName: "pl-[18px]",
        render: (identity) => {
          const isNavigating = identity.id === navigatingIdentityId;
          const truncatedExternalId =
            identity.externalId.length > 50
              ? `${identity.externalId.slice(0, 50)}...`
              : identity.externalId;

          const iconContainer = (
            <div className={cn("size-5 rounded flex items-center justify-center", "bg-brandA-3")}>
              {isNavigating ? (
                <div className="text-brandA-11">
                  <Loading size={18} />
                </div>
              ) : (
                <Fingerprint iconSize="md-medium" className="text-brandA-11" />
              )}
            </div>
          );

          return (
            <div className="flex flex-col items-start px-[18px] py-[6px]">
              <div className="flex gap-4 items-center">
                {iconContainer}
                <div className="flex flex-col gap-1 text-xs">
                  <span
                    className="font-sans text-accent-12 font-medium truncate"
                    title={identity.externalId}
                  >
                    {truncatedExternalId}
                  </span>
                  <InfoTooltip
                    content={
                      <div className="inline-flex justify-center gap-3 items-center font-mono text-xs text-gray-11">
                        <span>{identity.id}</span>
                        <CopyButton value={identity.id} />
                      </div>
                    }
                    position={{ side: "bottom", align: "start" }}
                  >
                    <Link
                      className="font-mono group-hover:underline decoration-dotted text-accent-9"
                      href={`/${workspace.slug}/identities/${identity.id}`}
                      aria-disabled={isNavigating}
                      onClick={() => {
                        handleLinkClick(identity.id);
                      }}
                    >
                      {shortenId(identity.id)}
                    </Link>
                  </InfoTooltip>
                </div>
              </div>
            </div>
          );
        },
      },
      {
        key: "keys",
        header: "Keys",
        width: "10%",
        render: (identity) => (
          <div className="flex items-center px-3 py-1">
            <span className="text-xs text-accent-11">{identity.keys.length}</span>
          </div>
        ),
      },
      {
        key: "ratelimits",
        header: "Ratelimits",
        width: "10%",
        render: (identity) => (
          <div className="flex items-center px-3 py-1">
            <span className="text-xs text-accent-11">{identity.ratelimits.length}</span>
          </div>
        ),
      },
      {
        key: "created",
        header: "Created",
        width: "15%",
        render: (identity) => {
          const date = new Date(identity.createdAt);
          const formatted = date.toLocaleDateString("en-US", {
            year: "numeric",
            month: "short",
            day: "numeric",
          });
          return (
            <div className="flex items-center px-3 py-1">
              <span className="text-xs text-accent-11">{formatted}</span>
            </div>
          );
        },
      },
      {
        key: "last_used",
        header: "Last Used",
        width: "15%",
        render: (identity) => (
          <div className="flex items-center px-3 py-1">
            <LastUsedCell identityId={identity.id} />
          </div>
        ),
      },
      {
        key: "action",
        header: "",
        width: "15%",
        render: (identity) => (
          <div className="flex items-center justify-end px-3 py-1">
            <IdentityTableActionPopover identity={identity} />
          </div>
        ),
      },
    ],
    [workspace.slug, navigatingIdentityId, handleLinkClick],
  );

  return (
    <VirtualTable
      data={identitiesList}
      columns={columns}
      isLoading={isLoadingInitial}
      isFetchingNextPage={isFetchingNextPage}
      onLoadMore={fetchNextPage}
      keyExtractor={(identity) => identity.id}
      onRowClick={handleRowClick}
      selectedItem={selectedIdentity}
      rowClassName={(identity) =>
        cn(
          "hover:bg-gray-2 transition-colors cursor-pointer",
          selectedIdentity?.id === identity.id && "bg-gray-3",
        )
      }
      loadMoreFooterProps={{
        hide: isLoadingInitial,
        buttonText: "Load more identities",
        hasMore: hasNextPage,
        countInfoText: (
          <div className="flex gap-2">
            <span>Showing</span>{" "}
            <span className="text-accent-12">
              {new Intl.NumberFormat().format(identitiesList.length)}
            </span>
            <span>of</span>
            <span className="text-accent-12">
              {new Intl.NumberFormat().format(totalCount)}
            </span>
            <span>identities</span>
          </div>
        ),
      }}
      emptyState={
        <div className="w-full flex justify-center items-center h-full">
          <Empty className="w-[400px] flex items-start">
            <Empty.Icon className="w-auto" />
            <Empty.Title>No Identities Found</Empty.Title>
            <Empty.Description className="text-left">
              {search
                ? "Try adjusting your search query"
                : "There are no identities yet. Create your first identity to get started."}
            </Empty.Description>
            <Empty.Actions className="mt-4 justify-start">
              <a
                href="https://www.unkey.com/docs/concepts/identities/overview"
                target="_blank"
                rel="noopener noreferrer"
              >
                <Button size="md">
                  <BookBookmark />
                  Learn about Identities
                </Button>
              </a>
            </Empty.Actions>
          </Empty>
        </div>
      }
      config={{
        rowHeight: 52,
        layoutMode: "grid",
        rowBorders: true,
        containerPadding: "px-0",
      }}
      renderSkeletonRow={({ columns, rowHeight }) =>
        columns.map((column, idx) => (
          <td
            key={column.key}
            className={cn(
              "text-xs align-middle whitespace-nowrap pr-4",
              idx === 0 ? "pl-[18px]" : "",
              column.key === "externalId" ? "py-[6px]" : "py-1",
            )}
            style={{ height: `${rowHeight}px` }}
          >
            {column.key === "externalId" && <IdentityColumnSkeleton />}
            {column.key === "keys" && <CountColumnSkeleton />}
            {column.key === "ratelimits" && <CountColumnSkeleton />}
            {column.key === "created" && <CreatedColumnSkeleton />}
            {column.key === "last_used" && <LastUsedColumnSkeleton />}
            {column.key === "action" && <ActionColumnSkeleton />}
            {!["externalId", "keys", "ratelimits", "created", "last_used", "action"].includes(
              column.key,
            ) && <div className="h-4 w-full bg-grayA-3 rounded animate-pulse" />}
          </td>
        ))
      }
    />
  );
};
