"use client";

import type { IdentityResponseSchema } from "@/lib/trpc/routers/identity/query";
import type { DataTableColumnDef } from "@unkey/ui";
import { SortableHeader, TimestampCell } from "@unkey/ui";
import dynamic from "next/dynamic";
import type { z } from "zod";
import { IdentityExternalIdCell } from "../components/cells/identity-external-id-cell";
import { IdentityLastUsedCell } from "../components/cells/identity-last-used-cell";

type Identity = z.infer<typeof IdentityResponseSchema>;

const IdentityTableActionPopover = dynamic(
  () =>
    import("@/components/identities-table/components/identity-table-actions").then(
      (mod) => mod.IdentityTableActions,
    ),
  { ssr: false },
);

export const IDENTITY_COLUMN_IDS = {
  EXTERNAL_ID: "externalId",
  KEYS: "keys",
  RATELIMITS: "ratelimits",
  CREATED: "created",
  LAST_USED: "last_used",
  ACTION: "action",
} as const;

type CreateIdentitiesColumnsOptions = {
  workspaceSlug: string;
};

export const createIdentitiesColumns = ({
  workspaceSlug,
}: CreateIdentitiesColumnsOptions): DataTableColumnDef<Identity>[] => [
  {
    id: IDENTITY_COLUMN_IDS.EXTERNAL_ID,
    accessorKey: "externalId",
    header: ({ header }) => <SortableHeader header={header}>External ID</SortableHeader>,
    enableSorting: true,
    meta: {
      width: "20%",
      headerClassName: "pl-[18px]",
    },
    cell: ({ row }) => (
      <IdentityExternalIdCell identity={row.original} workspaceSlug={workspaceSlug} />
    ),
  },
  {
    id: IDENTITY_COLUMN_IDS.KEYS,
    accessorFn: (row) => row.keys.length,
    header: ({ header }) => <SortableHeader header={header}>Keys</SortableHeader>,
    enableSorting: true,
    meta: {
      width: "10%",
    },
    cell: ({ row }) => (
      <div className="flex items-center px-3 py-1">
        <span className="text-xs text-accent-11">{row.original.keys.length}</span>
      </div>
    ),
  },
  {
    id: IDENTITY_COLUMN_IDS.RATELIMITS,
    accessorFn: (row) => row.ratelimits.length,
    header: ({ header }) => <SortableHeader header={header}>Ratelimits</SortableHeader>,
    enableSorting: true,
    meta: {
      width: "10%",
    },
    cell: ({ row }) => (
      <div className="flex items-center px-3 py-1">
        <span className="text-xs text-accent-11">{row.original.ratelimits.length}</span>
      </div>
    ),
  },
  {
    id: IDENTITY_COLUMN_IDS.CREATED,
    accessorKey: "createdAt",
    sortDescFirst: true,
    header: ({ header }) => <SortableHeader header={header}>Created</SortableHeader>,
    enableSorting: true,
    meta: {
      width: "15%",
    },
    cell: ({ row }) => (
      <div className="flex items-center px-3 py-1">
        <TimestampCell timestamp={row.original.createdAt} format="absolute" />
      </div>
    ),
  },
  {
    id: IDENTITY_COLUMN_IDS.LAST_USED,
    // Dummy accessor required for TanStack getCanSort() — actual sorting is server-side (manualSorting).
    accessorFn: () => 0,
    sortDescFirst: true,
    header: ({ header }) => <SortableHeader header={header}>Last Used</SortableHeader>,
    enableSorting: true,
    meta: {
      width: "15%",
    },
    cell: ({ row }) => (
      <div className="flex items-center px-3 py-1">
        <IdentityLastUsedCell identityId={row.original.id} isSelected={row.getIsSelected()} />
      </div>
    ),
  },
  {
    id: IDENTITY_COLUMN_IDS.ACTION,
    header: "",
    enableSorting: false,
    meta: {
      width: "15%",
    },
    cell: ({ row }) => (
      <div
        className="flex items-center justify-end px-3 py-1"
        onClick={(e) => e.stopPropagation()}
        onKeyDown={(e) => {
          if (e.key === "Enter" || e.key === " ") {
            e.stopPropagation();
          }
        }}
      >
        <IdentityTableActionPopover identity={row.original} />
      </div>
    ),
  },
];
