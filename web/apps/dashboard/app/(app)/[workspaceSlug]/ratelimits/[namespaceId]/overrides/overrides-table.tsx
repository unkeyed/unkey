"use client";
import { createOverridesColumns, renderOverridesSkeletonRow } from "@/components/overrides-table";
import { type RatelimitOverride, collection } from "@/lib/collections";
import { eq, useLiveQuery } from "@tanstack/react-db";
import {
  DataTable,
  type DataTableConfig,
  Empty,
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

// Overrides are backed by a TanStack DB live collection rather than a paginated
// tRPC query: the backend returns every override for the workspace at once and
// the collection stays reactive to local insert/update/delete. Since the full
// set already lives in memory we paginate on the client — slicing into pages of
// PAGE_SIZE and driving navigation with PaginationFooter — rather than
// round-tripping per page. Sorting stays disabled.
export const OverridesTable = ({ namespaceId }: Props) => {
  const [selectedOverride, setSelectedOverride] = useState<RatelimitOverride | null>(null);
  const [page, setPage] = useState(1);

  const { data: overrides, isLoading } = useLiveQuery((q) =>
    q
      .from({ override: collection.ratelimitOverrides })
      .where(({ override }) => eq(override.namespaceId, namespaceId)),
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
          <div className="w-full flex justify-center items-center h-full">
            <Empty className="w-[400px] flex items-start">
              <Empty.Icon className="w-auto" />
              <Empty.Title>No overrides found</Empty.Title>
              <Empty.Description className="text-left">
                No custom ratelimits found. Create your first override to get started.
              </Empty.Description>
            </Empty>
          </div>
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
      {selectedOverride && (
        <IdentifierDialog
          isModalOpen={!!selectedOverride}
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
