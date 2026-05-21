"use client";
import { createOverridesColumns, renderOverridesSkeletonRow } from "@/components/overrides-table";
import { type RatelimitOverride, collection } from "@/lib/collections";
import { eq, useLiveQuery } from "@tanstack/react-db";
import { DataTable, Empty, getSelectableRowClassName } from "@unkey/ui";
import { useMemo, useState } from "react";
import { IdentifierDialog } from "../_components/identifier-dialog";

type Props = {
  namespaceId: string;
};

// Overrides are backed by a TanStack DB live collection rather than a paginated
// tRPC query: the backend returns every override for the workspace at once and
// the collection stays reactive to local insert/update/delete. There is no
// server pagination or sorting, so the DataTable renders the full (virtualized)
// list with sorting disabled and no pagination footer.
export const OverridesTable = ({ namespaceId }: Props) => {
  const [selectedOverride, setSelectedOverride] = useState<RatelimitOverride | null>(null);
  const [isDialogOpen, setIsDialogOpen] = useState(false);

  const { data: overrides, isLoading } = useLiveQuery((q) =>
    q
      .from({ override: collection.ratelimitOverrides })
      .where(({ override }) => eq(override.namespaceId, namespaceId)),
  );

  const handleRowClick = (override: RatelimitOverride | null) => {
    if (!override) {
      return;
    }
    setSelectedOverride(override);
    setIsDialogOpen(true);
  };

  const columns = useMemo(() => createOverridesColumns({ namespaceId }), [namespaceId]);

  return (
    <>
      <DataTable
        data={overrides}
        columns={columns}
        getRowId={(override) => override.id}
        isLoading={isLoading}
        enableSorting={false}
        onRowClick={handleRowClick}
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
        config={{ rowHeight: 52, layout: "grid", rowBorders: true, containerPadding: "px-0" }}
      />
      {selectedOverride && (
        <IdentifierDialog
          isModalOpen={isDialogOpen}
          onOpenChange={setIsDialogOpen}
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
