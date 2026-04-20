"use client";

import {
  createIdentitiesColumns,
  getRowClassName,
  renderIdentitiesSkeletonRow,
  useIdentitiesQuery,
} from "@/components/identities-table";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import type { IdentityResponseSchema } from "@/lib/trpc/routers/identity/query";
import { BookBookmark } from "@unkey/icons";
import { Button, DataTable, Empty, PaginationFooter } from "@unkey/ui";
import { useRouter } from "next/navigation";
import { parseAsString, useQueryState } from "nuqs";
import { useCallback, useMemo, useState } from "react";
import type { z } from "zod";

type Identity = z.infer<typeof IdentityResponseSchema>;

const TABLE_CONFIG = {
  rowHeight: 52,
  layout: "grid" as const,
  rowBorders: true,
  containerPadding: "px-0",
};

export const IdentitiesList = () => {
  const workspace = useWorkspaceNavigation();
  const router = useRouter();
  const [selectedIdentity, setSelectedIdentity] = useState<Identity | null>(null);
  const [search] = useQueryState(
    "search",
    parseAsString.withDefault("").withOptions({
      history: "replace",
      shallow: true,
      clearOnDefault: true,
    }),
  );

  const {
    identities,
    isInitialLoading,
    isFetching,
    page,
    pageSize,
    totalPages,
    totalCount,
    onPageChange,
    sorting,
    onSortingChange,
  } = useIdentitiesQuery();

  const handleRowClick = useCallback(
    (identity: Identity | null) => {
      if (identity) {
        setSelectedIdentity(identity);
        router.push(`/${workspace.slug}/identities/${identity.id}`);
      } else {
        setSelectedIdentity(null);
      }
    },
    [router, workspace.slug],
  );

  const getRowClassNameMemoized = useCallback(
    (identity: Identity) => getRowClassName(identity, selectedIdentity),
    [selectedIdentity],
  );

  const columns = useMemo(
    () => createIdentitiesColumns({ workspaceSlug: workspace.slug }),
    [workspace.slug],
  );

  const isNavigating = isFetching && !isInitialLoading;

  return (
    <>
      <DataTable
        data={identities}
        columns={columns}
        getRowId={(identity) => identity.id}
        isLoading={isInitialLoading}
        enableSorting={true}
        manualSorting={true}
        sorting={sorting}
        onSortingChange={onSortingChange}
        onRowClick={handleRowClick}
        selectedItem={selectedIdentity}
        rowClassName={getRowClassNameMemoized}
        renderSkeletonRow={renderIdentitiesSkeletonRow}
        config={TABLE_CONFIG}
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
      />
      <PaginationFooter
        page={page}
        pageSize={pageSize}
        totalPages={totalPages}
        totalCount={totalCount}
        onPageChange={onPageChange}
        itemLabel="identities"
        loading={isInitialLoading}
        disabled={isNavigating}
      />
    </>
  );
};
