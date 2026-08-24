import type { RootKey } from "@/lib/trpc/routers/settings/root-keys/query";
import { cn } from "@/lib/utils";
import type { DataTableColumnDef } from "@unkey/ui";
import { LastUpdatedCell, RootKeyNameCell, RowActionSkeleton, SortableHeader } from "@unkey/ui";
import { TimestampInfo } from "@unkey/ui";
import dynamic from "next/dynamic";
import { PermissionsCell } from "../components/settings-root-keys/permissions-cell";

const RootKeysTableActions = dynamic(
  () =>
    import("../components/settings-root-keys/root-keys-table-action.popover").then(
      (mod) => mod.RootKeysTableActions,
    ),
  {
    loading: () => <RowActionSkeleton />,
  },
);

export const ROOT_KEY_COLUMN_IDS = {
  ROOT_KEY: { id: "root_key", accessorKey: "name", header: "Name" },
  PERMISSIONS: { id: "permissions", accessorKey: "permissions", header: "Permissions" },
  CREATED_AT: { id: "created_at", accessorKey: "created_at", header: "Created At" },
  LAST_UPDATED: { id: "last_updated", accessorKey: "last_updated", header: "Last Updated" },
  ACTION: { id: "action", accessorKey: "action", header: "Action" },
} as const;

type CreateRootKeyColumnsOptions = {
  selectedRootKeyId?: string;
  onEditKey: (rootKey: RootKey) => void;
};

export const createRootKeyColumns = ({
  selectedRootKeyId,
  onEditKey,
}: CreateRootKeyColumnsOptions): DataTableColumnDef<RootKey>[] => [
  {
    id: ROOT_KEY_COLUMN_IDS.ROOT_KEY.id,
    accessorKey: ROOT_KEY_COLUMN_IDS.ROOT_KEY.accessorKey,
    header: ({ header }) => (
      <SortableHeader key={ROOT_KEY_COLUMN_IDS.ROOT_KEY.id} header={header}>
        {ROOT_KEY_COLUMN_IDS.ROOT_KEY.header}
      </SortableHeader>
    ),
    meta: {
      width: {
        min: 170,
        max: 400,
      },
      headerClassName: "pl-[18px]",
    },
    cell: ({ row }) => {
      const rootKey = row.original;
      const isSelected = rootKey.id === selectedRootKeyId;
      return <RootKeyNameCell name={rootKey.name ?? undefined} isSelected={isSelected} />;
    },
  },
  {
    id: ROOT_KEY_COLUMN_IDS.PERMISSIONS.id,
    header: ROOT_KEY_COLUMN_IDS.PERMISSIONS.header,
    enableSorting: false,
    meta: {
      width: {
        min: 170,
        max: 400,
      },
    },
    cell: ({ row }) => {
      const rootKey = row.original;
      return (
        <PermissionsCell
          permissions={rootKey.permissions}
          isSelected={rootKey.id === selectedRootKeyId}
        />
      );
    },
  },
  {
    id: ROOT_KEY_COLUMN_IDS.CREATED_AT.id,
    accessorKey: ROOT_KEY_COLUMN_IDS.CREATED_AT.accessorKey,
    sortDescFirst: true,
    header: ({ header }) => (
      <SortableHeader key={ROOT_KEY_COLUMN_IDS.CREATED_AT.id} header={header}>
        {ROOT_KEY_COLUMN_IDS.CREATED_AT.header}
      </SortableHeader>
    ),
    meta: {
      width: {
        min: 140,
        max: 300,
      },
    },
    cell: ({ row }) => {
      const rootKey = row.original;
      return (
        <TimestampInfo
          value={rootKey.createdAt}
          className={cn("font-mono group-hover:underline decoration-dotted")}
        />
      );
    },
  },
  {
    id: ROOT_KEY_COLUMN_IDS.LAST_UPDATED.id,
    accessorKey: ROOT_KEY_COLUMN_IDS.LAST_UPDATED.accessorKey,
    header: ({ header }) => (
      <SortableHeader key={ROOT_KEY_COLUMN_IDS.LAST_UPDATED.id} header={header}>
        {ROOT_KEY_COLUMN_IDS.LAST_UPDATED.header}
      </SortableHeader>
    ),
    meta: {
      width: {
        min: 140,
        max: 300,
      },
    },
    cell: ({ row }) => {
      const rootKey = row.original;
      return (
        <LastUpdatedCell
          lastUpdated={rootKey.lastUpdatedAt ?? 0}
          isSelected={rootKey.id === selectedRootKeyId}
        />
      );
    },
  },
  {
    id: ROOT_KEY_COLUMN_IDS.ACTION.id,
    header: "",
    meta: {
      width: {
        min: 60,
        max: 100,
      },
    },
    cell: ({ row }) => {
      const rootKey = row.original;
      return <RootKeysTableActions rootKey={rootKey} onEditKey={onEditKey} />;
    },
  },
];
