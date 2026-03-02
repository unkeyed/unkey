import type { RootKey } from "@/lib/trpc/routers/settings/root-keys/query";
import { cn } from "@/lib/utils";
import type { DataTableColumnDef } from "@unkey/ui";
import {
  AssignedItemsCell,
  HiddenValueCell,
  LastUpdatedCell,
  RootKeyNameCell,
  RowActionSkeleton,
  SortableHeader,
} from "@unkey/ui";
import { InfoTooltip, TimestampInfo } from "@unkey/ui";
import dynamic from "next/dynamic";

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
  ROOT_KEY: "root_key",
  KEY: "key",
  PERMISSIONS: "permissions",
  CREATED_AT: "created_at",
  LAST_UPDATED: "last_updated",
  ACTION: "action",
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
    id: ROOT_KEY_COLUMN_IDS.ROOT_KEY,
    accessorKey: "name",
    header: ({ header }) => (
      <SortableHeader key={ROOT_KEY_COLUMN_IDS.ROOT_KEY} header={header}>
        Name
      </SortableHeader>
    ),
    meta: {
      width: "20%",
      headerClassName: "pl-[18px]",
    },
    sortingFn: "alphanumeric",
    cell: ({ row }) => {
      const rootKey = row.original;
      const isSelected = rootKey.id === selectedRootKeyId;
      return <RootKeyNameCell name={rootKey.name ?? undefined} isSelected={isSelected} />;
    },
  },
  {
    id: ROOT_KEY_COLUMN_IDS.KEY,
    accessorKey: "start",
    header: "Key",
    meta: {
      width: "18%",
    },
    sortingFn: "alphanumeric",
    cell: ({ row }) => {
      const rootKey = row.original;
      return (
        <InfoTooltip
          content={
            <p>
              This is the first part of the key to visually match it. We don&apos;t store the full
              key for security reasons.
            </p>
          }
        >
          <HiddenValueCell
            value={rootKey.start}
            title="Key"
            selected={selectedRootKeyId === rootKey.id}
          />
        </InfoTooltip>
      );
    },
  },
  {
    id: ROOT_KEY_COLUMN_IDS.PERMISSIONS,
    header: "Permissions",
    meta: {
      width: "15%",
    },
    cell: ({ row }) => {
      const rootKey = row.original;
      return (
        <AssignedItemsCell
          isSelected={rootKey.id === selectedRootKeyId}
          permissionSummary={rootKey.permissionSummary}
        />
      );
    },
  },
  {
    id: ROOT_KEY_COLUMN_IDS.CREATED_AT,
    accessorKey: "createdAt",
    header: ({ header }) => (
      <SortableHeader key={ROOT_KEY_COLUMN_IDS.CREATED_AT} header={header}>
        Created At
      </SortableHeader>
    ),
    meta: {
      width: "14%",
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
    id: ROOT_KEY_COLUMN_IDS.LAST_UPDATED,
    accessorKey: "lastUpdatedAt",
    header: ({ header }) => (
      <SortableHeader key={ROOT_KEY_COLUMN_IDS.LAST_UPDATED} header={header}>
        Last Updated
      </SortableHeader>
    ),
    meta: {
      width: "20%",
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
    id: ROOT_KEY_COLUMN_IDS.ACTION,
    header: "",
    meta: {
      width: "auto",
    },
    cell: ({ row }) => {
      const rootKey = row.original;
      return <RootKeysTableActions rootKey={rootKey} onEditKey={onEditKey} />;
    },
  },
];
