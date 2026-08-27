import type { RootKey } from "@/lib/trpc/routers/settings/root-keys/query";
import type { DataTableColumnDef } from "@unkey/ui";
import { RootKeyNameCell, RowActionSkeleton, TimestampInfo } from "@unkey/ui";
import dynamic from "next/dynamic";
import { PermissionsCell } from "./permissions-cell";

const RootKeysTableActions = dynamic(
  () => import("./root-keys-table-action.popover").then((mod) => mod.RootKeysTableActions),
  {
    loading: () => <RowActionSkeleton />,
    ssr: false,
  },
);

export const ROOT_KEY_COLUMN_IDS = {
  ROOT_KEY: { id: "root_key", accessorKey: "name", header: "Name" },
  PERMISSIONS: { id: "permissions", header: "Permissions" },
  CREATED_AT: { id: "created_at", accessorKey: "createdAt", header: "Created At" },
  ACTION: { id: "action", header: "" },
} as const;

type CreateRootKeyColumnsOptions = {
  onEditKey: (rootKey: RootKey) => void;
};

export const createRootKeyColumns = ({
  onEditKey,
}: CreateRootKeyColumnsOptions): DataTableColumnDef<RootKey>[] => [
  {
    id: ROOT_KEY_COLUMN_IDS.ROOT_KEY.id,
    accessorKey: ROOT_KEY_COLUMN_IDS.ROOT_KEY.accessorKey,
    header: ROOT_KEY_COLUMN_IDS.ROOT_KEY.header,
    enableSorting: false,
    meta: {
      width: { min: 170, max: 400 },
      headerClassName: "pl-[18px]",
    },
    cell: ({ row }) => <RootKeyNameCell name={row.original.name ?? undefined} />,
  },
  {
    id: ROOT_KEY_COLUMN_IDS.PERMISSIONS.id,
    header: ROOT_KEY_COLUMN_IDS.PERMISSIONS.header,
    enableSorting: false,
    meta: {
      width: { min: 170, max: 400 },
    },
    cell: ({ row }) => <PermissionsCell permissions={row.original.permissions} />,
  },
  {
    id: ROOT_KEY_COLUMN_IDS.CREATED_AT.id,
    accessorKey: ROOT_KEY_COLUMN_IDS.CREATED_AT.accessorKey,
    header: ROOT_KEY_COLUMN_IDS.CREATED_AT.header,
    enableSorting: false,
    meta: {
      width: { min: 140, max: 300 },
    },
    cell: ({ row }) => (
      <TimestampInfo
        value={row.original.createdAt}
        displayType="relative"
        side="top"
        align="center"
        className="text-[13px] text-gray-9"
      />
    ),
  },
  {
    id: ROOT_KEY_COLUMN_IDS.ACTION.id,
    header: ROOT_KEY_COLUMN_IDS.ACTION.header,
    enableSorting: false,
    meta: {
      width: { min: 60, max: 100 },
    },
    cell: ({ row }) => <RootKeysTableActions rootKey={row.original} onEditKey={onEditKey} />,
  },
];
