import type { RoleBasic } from "@/lib/trpc/routers/authorization/roles/query";
import { cn } from "@/lib/utils";
import { Key2, Page2, Tag } from "@unkey/icons";
import type { DataTableColumnDef } from "@unkey/ui";
import { Checkbox, LastUpdatedCell, RowActionSkeleton, SortableHeader } from "@unkey/ui";
import dynamic from "next/dynamic";

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
      width: "20%",
      headerClassName: "pl-[18px]",
    },
    cell: ({ row }) => {
      const role = row.original;
      const isSelected = selectedRoles.has(role.roleId);
      const isHovered = hoveredRoleName === role.name;

      const iconContainer = (
        <div
          className={cn(
            "size-5 rounded-sm flex items-center justify-center border border-grayA-3 transition-all duration-100",
            "bg-grayA-3",
            isSelected && "bg-grayA-5",
          )}
          onMouseEnter={() => onHoverRole(role.name)}
          onMouseLeave={() => onHoverRole(null)}
        >
          {!isSelected && !isHovered && (
            <Tag
              iconSize="sm-regular"
              className="text-gray-12 cursor-pointer w-[12px] h-[12px]"
            />
          )}
          {(isSelected || isHovered) && (
            <Checkbox
              checked={isSelected}
              className="size-4 [&_svg]:size-3"
              onClick={(e) => e.stopPropagation()}
              onCheckedChange={() => onToggleSelection(role.roleId)}
            />
          )}
        </div>
      );

      return (
        <div className="flex flex-col items-start px-[18px] py-[6px]">
          <div className="flex gap-4 items-center">
            {iconContainer}
            <div className="flex flex-col gap-1 text-xs">
              <div className="font-medium truncate text-accent-12 leading-4 text-[13px] max-w-[120px]">
                {role.name}
              </div>
              {role.description ? (
                <span
                  className="font-sans text-accent-9 truncate max-w-[180px] text-xs"
                  title={role.description}
                >
                  {role.description}
                </span>
              ) : (
                <span className="font-sans text-accent-9 truncate max-w-[180px] text-xs italic">
                  No description
                </span>
              )}
            </div>
          </div>
        </div>
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
      width: "20%",
    },
    cell: ({ row }) => {
      const role = row.original;
      const isSelected = role.roleId === selectedRoleId;
      const count = role.assignedKeys;

      const icon = <Key2 iconSize="md-medium" className="opacity-50" />;

      if (count === 0) {
        return (
          <div className="flex flex-col gap-1 py-2 max-w-[200px]">
            <div
              className={cn(
                "rounded-md py-[2px] px-1.5 items-center w-fit flex gap-2 transition-all duration-100 border border-dashed bg-grayA-2",
                isSelected
                  ? "border-grayA-7 text-grayA-9"
                  : "border-grayA-6 text-grayA-8",
              )}
            >
              {icon}
              <span className="text-grayA-9 text-xs">None assigned</span>
            </div>
          </div>
        );
      }

      return (
        <div className="flex flex-col gap-1 py-2 max-w-[200px]">
          <div
            className={cn(
              "font-mono rounded-md py-[2px] px-1.5 items-center w-fit flex gap-2 transition-all duration-100 border border-dashed text-grayA-12",
              isSelected
                ? "bg-grayA-4 border-grayA-7"
                : "bg-grayA-3 border-grayA-6 group-hover:bg-grayA-4",
            )}
          >
            {icon}
            <div className="text-grayA-11 text-xs max-w-[150px] truncate">
              {count} {count === 1 ? "Key" : "Keys"}
            </div>
          </div>
        </div>
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
      width: "20%",
    },
    cell: ({ row }) => {
      const role = row.original;
      const isSelected = role.roleId === selectedRoleId;
      const count = role.assignedPermissions;

      const icon = <Page2 iconSize="md-medium" className="opacity-50" />;

      if (count === 0) {
        return (
          <div className="flex flex-col gap-1 py-2 max-w-[200px]">
            <div
              className={cn(
                "rounded-md py-[2px] px-1.5 items-center w-fit flex gap-2 transition-all duration-100 border border-dashed bg-grayA-2",
                isSelected
                  ? "border-grayA-7 text-grayA-9"
                  : "border-grayA-6 text-grayA-8",
              )}
            >
              {icon}
              <span className="text-grayA-9 text-xs">None assigned</span>
            </div>
          </div>
        );
      }

      return (
        <div className="flex flex-col gap-1 py-2 max-w-[200px]">
          <div
            className={cn(
              "font-mono rounded-md py-[2px] px-1.5 items-center w-fit flex gap-2 transition-all duration-100 border border-dashed text-grayA-12",
              isSelected
                ? "bg-grayA-4 border-grayA-7"
                : "bg-grayA-3 border-grayA-6 group-hover:bg-grayA-4",
            )}
          >
            {icon}
            <div className="text-grayA-11 text-xs max-w-[150px] truncate">
              {count} {count === 1 ? "Permission" : "Permissions"}
            </div>
          </div>
        </div>
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
      width: "12%",
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
      width: "auto",
    },
    cell: ({ row }) => {
      const role = row.original;
      return <RolesTableActions role={role} />;
    },
  },
];
