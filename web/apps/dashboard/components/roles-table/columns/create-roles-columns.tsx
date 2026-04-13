import type { RoleBasic } from "@/lib/trpc/routers/authorization/roles/query";
import { Key2, Page2 } from "@unkey/icons";
import type { DataTableColumnDef } from "@unkey/ui";
import { AssignedCountCell, LastUpdatedCell, RowActionSkeleton, SortableHeader } from "@unkey/ui";
import dynamic from "next/dynamic";
import { RoleNameCell } from "../components/cells/role-name-cell";

const RolesTableActions = dynamic(
  () =>
    import(
      "@/app/(app)/[workspaceSlug]/authorization/roles/components/table/components/actions/keys-table-action.popover.constants"
    ).then((mod) => mod.RolesTableActions),
  {
    loading: () => <RowActionSkeleton />,
  },
);

export const ROLE_COLUMN_IDS = {
  ROLE: { id: "role", accessorKey: "name", header: "Role" },
  ASSIGNED_KEYS: { id: "assignedKeys", accessorKey: "assignedKeys", header: "Assigned Keys" },
  PERMISSIONS: {
    id: "permissions",
    accessorKey: "assignedPermissions",
    header: "Assigned Permissions",
  },
  LAST_UPDATED: { id: "last_updated", accessorKey: "lastUpdated", header: "Last Updated" },
  ACTION: { id: "action", accessorKey: "action", header: "" },
} as const;

type CreateRolesColumnsOptions = {
  selectedRoleId?: string;
  selectedRoles: Set<string>;
  hoveredRoleName: string | null;
  onToggleSelection: (roleId: string) => void;
  onHoverRole: (name: string | null) => void;
};

export const createRolesColumns = ({
  selectedRoleId,
  selectedRoles,
  hoveredRoleName,
  onToggleSelection,
  onHoverRole,
}: CreateRolesColumnsOptions): DataTableColumnDef<RoleBasic>[] => [
  {
    id: ROLE_COLUMN_IDS.ROLE.id,
    accessorKey: ROLE_COLUMN_IDS.ROLE.accessorKey,
    header: ({ header }) => (
      <SortableHeader key={ROLE_COLUMN_IDS.ROLE.id} header={header}>
        {ROLE_COLUMN_IDS.ROLE.header}
      </SortableHeader>
    ),
    meta: {
      width: {
        min: 260,
        max: 400,
      },
      headerClassName: "pl-[18px]",
    },
    cell: ({ row }) => {
      const role = row.original;
      return (
        <RoleNameCell
          name={role.name}
          description={role.description}
          isSelected={selectedRoles.has(role.roleId)}
          isHovered={hoveredRoleName === role.name}
          onMouseEnter={() => onHoverRole(role.name)}
          onMouseLeave={() => onHoverRole(null)}
          onCheckedChange={() => onToggleSelection(role.roleId)}
        />
      );
    },
  },
  {
    id: ROLE_COLUMN_IDS.ASSIGNED_KEYS.id,
    accessorKey: ROLE_COLUMN_IDS.ASSIGNED_KEYS.accessorKey,
    sortDescFirst: true,
    header: ({ header }) => (
      <SortableHeader key={ROLE_COLUMN_IDS.ASSIGNED_KEYS.id} header={header}>
        {ROLE_COLUMN_IDS.ASSIGNED_KEYS.header}
      </SortableHeader>
    ),
    meta: {
      width: {
        min: 160,
        max: 400,
      },
    },
    cell: ({ row }) => {
      const role = row.original;
      return (
        <AssignedCountCell
          count={role.assignedKeys}
          icon={<Key2 iconSize="md-medium" className="opacity-50" />}
          singularLabel="Key"
          isSelected={role.roleId === selectedRoleId}
        />
      );
    },
  },
  {
    id: ROLE_COLUMN_IDS.PERMISSIONS.id,
    accessorKey: ROLE_COLUMN_IDS.PERMISSIONS.accessorKey,
    sortDescFirst: true,
    header: ({ header }) => (
      <SortableHeader key={ROLE_COLUMN_IDS.PERMISSIONS.id} header={header}>
        {ROLE_COLUMN_IDS.PERMISSIONS.header}
      </SortableHeader>
    ),
    meta: {
      width: {
        min: 200,
        max: 400,
      },
    },
    cell: ({ row }) => {
      const role = row.original;
      return (
        <AssignedCountCell
          count={role.assignedPermissions}
          icon={<Page2 iconSize="md-medium" className="opacity-50" />}
          singularLabel="Permission"
          isSelected={role.roleId === selectedRoleId}
        />
      );
    },
  },
  {
    id: ROLE_COLUMN_IDS.LAST_UPDATED.id,
    accessorKey: ROLE_COLUMN_IDS.LAST_UPDATED.accessorKey,
    sortDescFirst: true,
    header: ({ header }) => (
      <SortableHeader key={ROLE_COLUMN_IDS.LAST_UPDATED.id} header={header}>
        {ROLE_COLUMN_IDS.LAST_UPDATED.header}
      </SortableHeader>
    ),
    meta: {
      width: {
        min: 180,
        max: 400,
      },
    },
    cell: ({ row }) => {
      const role = row.original;
      return (
        <LastUpdatedCell
          lastUpdated={role.lastUpdated}
          isSelected={role.roleId === selectedRoleId}
        />
      );
    },
  },
  {
    id: ROLE_COLUMN_IDS.ACTION.id,
    header: "",
    enableSorting: false,
    meta: {
      width: {
        min: 60,
        max: 100,
      },
    },
    cell: ({ row }) => {
      const role = row.original;
      return <RolesTableActions role={role} />;
    },
  },
];
