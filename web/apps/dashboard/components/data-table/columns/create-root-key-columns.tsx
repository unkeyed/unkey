import { HiddenValueCell } from "@/components/data-table/components/cells/hidden-value-cell";
import type { DataTableColumnDef } from "@/components/data-table";
import { RowActionSkeleton } from "@/components/data-table/components/cells/row-action-skeleton";
import { RootKeyNameCell } from "@/components/data-table/components/cells/root-key-name-cell";
import { AssignedItemsCell } from "@/components/data-table/components/cells/assigned-items-cell";
import type { RootKey } from "@/lib/trpc/routers/settings/root-keys/query";
import { cn } from "@/lib/utils";
import { InfoTooltip, TimestampInfo } from "@unkey/ui";
import dynamic from "next/dynamic";
import { LastUpdatedCell } from "@/components/data-table/components/cells/last-updated-cell";

const RootKeysTableActions = dynamic(
  () =>
    import(
      "../components/settings-root-keys/root-keys-table-action.popover"
    ).then((mod) => mod.RootKeysTableActions),
  {
    loading: () => <RowActionSkeleton />,
  },
);

type CreateRootKeyColumnsOptions = {
  selectedRootKeyId?: string;
  onEditKey: (rootKey: RootKey) => void;
};

export const createRootKeyColumns = ({
  selectedRootKeyId,
  onEditKey,
}: CreateRootKeyColumnsOptions): DataTableColumnDef<RootKey>[] => [
  {
    id: "root_key",
    accessorKey: "name",
    header: "Name",
    meta: {
      width: "20%",
      headerClassName: "pl-[18px]",
    },
    cell: ({ row }) => {
      const rootKey = row.original;
      const isSelected = rootKey.id === selectedRootKeyId;
      return <RootKeyNameCell name={rootKey.name ?? undefined} isSelected={isSelected} />;
    },
  },
  {
    id: "key",
    accessorKey: "start",
    header: "Key",
    meta: {
      width: "18%",
    },
    cell: ({ row }) => {
      const rootKey = row.original;
      return (
        <InfoTooltip
          content={
            <p>
              This is the first part of the key to visually match it. We don't store the full key
              for security reasons.
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
    id: "permissions",
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
    id: "created_at",
    accessorKey: "createdAt",
    header: "Created At",
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
    id: "last_updated",
    accessorKey: "lastUpdatedAt",
    header: "Last Updated",
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
    id: "action",
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
