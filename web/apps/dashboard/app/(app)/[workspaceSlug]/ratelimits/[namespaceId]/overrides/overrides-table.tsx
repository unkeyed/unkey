"use client";
import { createOverridesColumns, renderOverridesSkeletonRow } from "@/components/overrides-table";
import { type RatelimitOverride, collection } from "@/lib/collections";
import { eq, useLiveQuery } from "@tanstack/react-db";
import { IconArrowDottedRotateAnticlockwiseOutline18 } from "@unkey/icons";
import {
  DataTable,
  type DataTableConfig,
  EmptyState,
  EmptyStateDescription,
  EmptyStateHeader,
  EmptyStateIcon,
  EmptyStateTitle,
  PaginationFooter,
  getSelectableRowClassName,
} from "@unkey/ui";
import { useMemo, useState } from "react";
import { IdentifierDialog } from "../_components/identifier-dialog";

type Props = {
  namespaceId: string;
};

// The original VirtualTable passed no config, so it rendered with the shared
// defaults: classic layout (4px spacing between rows), no row borders, and
// "px-2" container padding. DataTable's defaults match those exactly, so we
// only override the row height (26 vs the default 36) to preserve the look.
const TABLE_CONFIG: Partial<DataTableConfig> = {
  rowHeight: 26,
};

const PAGE_SIZE = 50;

// The collection loads every override of the namespace and stays reactive to
// local insert, update, and delete. The full set is in memory, so pages are
// slices of PAGE_SIZE on the client instead of a request per page. Sorting
// stays disabled.
export const OverridesTable = ({ namespaceId }: Props) => {
  const [selectedOverride, setSelectedOverride] = useState<RatelimitOverride | null>(null);
  const [page, setPage] = useState(1);

  const { data: overrides, isLoading } = useLiveQuery(
    (q) =>
      q
        .from({ override: collection.ratelimitOverrides })
        .where(({ override }) => eq(override.namespaceId, namespaceId)),
    [namespaceId],
  );

  const totalCount = overrides.length;
  const totalPages = Math.max(1, Math.ceil(totalCount / PAGE_SIZE));
  // Clamp so a shrinking collection (e.g. after a delete) never strands us on an
  // empty trailing page.
  const currentPage = Math.min(page, totalPages);

  const paginatedOverrides = useMemo(
    () => overrides.slice((currentPage - 1) * PAGE_SIZE, currentPage * PAGE_SIZE),
    [overrides, currentPage],
  );

  const columns = useMemo(() => createOverridesColumns({ namespaceId }), [namespaceId]);

  return (
    <>
      <DataTable
        data={paginatedOverrides}
        columns={columns}
        getRowId={(override) => override.id}
        isLoading={isLoading}
        enableSorting={false}
        onRowClick={setSelectedOverride}
        selectedItem={selectedOverride}
        rowClassName={(override) => getSelectableRowClassName(override.id === selectedOverride?.id)}
        renderSkeletonRow={renderOverridesSkeletonRow}
        emptyState={
          <EmptyState frame="none">
            <EmptyStateIcon>
              <IconArrowDottedRotateAnticlockwiseOutline18 />
            </EmptyStateIcon>
            <EmptyStateHeader>
              <EmptyStateTitle>No overrides found</EmptyStateTitle>
              <EmptyStateDescription>
                No custom ratelimits found. Create your first override to get started.
              </EmptyStateDescription>
            </EmptyStateHeader>
          </EmptyState>
        }
        config={TABLE_CONFIG}
      />
      <PaginationFooter
        page={currentPage}
        pageSize={PAGE_SIZE}
        totalPages={totalPages}
        totalCount={totalCount}
        onPageChange={setPage}
        itemLabel="overrides"
        loading={isLoading}
        hide={totalPages === 1}
      />
      {/* Conditionally mounted (rather than letting isModalOpen toggle visibility)
          so the dialog's form re-initializes its defaultValues from the clicked
          override — react-hook-form only reads defaults on mount. */}
      {selectedOverride && (
        <IdentifierDialog
          isModalOpen={true}
          onOpenChange={(open) => {
            if (!open) {
              setSelectedOverride(null);
            }
          }}
          namespaceId={namespaceId}
          identifier={selectedOverride.identifier}
          overrideDetails={{
            overrideId: selectedOverride.id,
            limit: selectedOverride.limit,
            duration: selectedOverride.duration,
          }}
        />
      )}
    </>
  );
};
