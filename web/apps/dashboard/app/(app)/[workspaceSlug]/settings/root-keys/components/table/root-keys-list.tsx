"use client";
import {
  createRootKeyColumns,
  getRowClassName,
  renderRootKeySkeletonRow,
  useRootKeysListPaginated,
} from "@/components/root-keys-table";
import { useRootKeysV2ListPaginated } from "@/components/root-keys-table/hooks/use-root-keys-v2-list-query";
import {
  type RootKeysTransport,
  RootKeysTransportProvider,
} from "@/components/root-keys-table/root-keys-transport";
import type { RootKey } from "@/lib/trpc/routers/settings/root-keys/query";
import { IconBookBookmarkOutline18 } from "@unkey/icons";
import type { UnkeyPermission } from "@unkey/rbac";
import { unkeyPermissionValidation } from "@unkey/rbac";
import {
  DataTable,
  EmptyState,
  EmptyStateActions,
  EmptyStateDescription,
  EmptyStateHeader,
  EmptyStateTitle,
  PaginationFooter,
  buttonVariants,
} from "@unkey/ui";
import { useCallback, useMemo, useState } from "react";
import { RootKeyDialog } from "../dialog/root-key-dialog";
import { V2RootKeyDialog } from "../dialog/v2-root-key-dialog";
import { type ListVariant, ListVariantDebugBar } from "./list-variant-debug-bar";
import { RootKeysResourceList } from "./root-keys-resource-list";

// Type guard function to check if a string is a valid UnkeyPermission
const isUnkeyPermission = (permissionName: string): permissionName is UnkeyPermission => {
  const result = unkeyPermissionValidation.safeParse(permissionName);
  return result.success;
};

const TABLE_CONFIG = {
  rowHeight: 40,
  layout: "grid" as const,
  rowBorders: true,
  containerPadding: "px-0",
};

type RootKeysListProps = { useV2: boolean };

export const RootKeysList = ({ useV2 }: RootKeysListProps) =>
  useV2 ? <V2RootKeysList /> : <LegacyRootKeysList />;

const LegacyRootKeysList = () => (
  <RootKeysListView transport="legacy" query={useRootKeysListPaginated()} />
);

const V2RootKeysList = () => (
  <RootKeysListView transport="v2" query={useRootKeysV2ListPaginated()} />
);

type RootKeysListViewProps = {
  transport: RootKeysTransport;
  query: ReturnType<typeof useRootKeysListPaginated>;
};

const RootKeysListView = ({ transport, query }: RootKeysListViewProps) => {
  const {
    rootKeys,
    isInitialLoading,
    isNavigating,
    totalCount,
    onPageChange,
    page,
    pageSize,
    totalPages,
    sorting,
    onSortingChange,
  } = query;
  const [variant, setVariant] = useState<ListVariant>("resource-list");
  const [selectedRootKey, setSelectedRootKey] = useState<RootKey | null>(null);
  const [editDialogOpen, setEditDialogOpen] = useState(false);
  const [editingKey, setEditingKey] = useState<RootKey | null>(null);

  const handleEditKey = useCallback((rootKey: RootKey) => {
    setEditingKey(rootKey);
    setEditDialogOpen(true);
  }, []);

  const handleEditDialogOpenChange = useCallback((open: boolean) => {
    setEditDialogOpen(open);
    if (!open) {
      setEditingKey(null);
    }
  }, []);

  const selectedRootKeyId = selectedRootKey?.id;

  const handleRowClick = useCallback((rootKey: RootKey | null) => {
    if (rootKey) {
      setEditingKey(rootKey);
      setSelectedRootKey(rootKey);
      setEditDialogOpen(true);
    } else {
      setSelectedRootKey(null);
    }
  }, []);

  const getRowClassNameMemoized = useCallback(
    (rootKey: RootKey) => getRowClassName(rootKey, selectedRootKey),
    [selectedRootKey],
  );

  const existingKey = useMemo(() => {
    if (!editingKey || transport === "v2") {
      return null;
    }

    // Guard against undefined permissions and use type guard function
    const permissions = editingKey.permissions ?? [];
    const validatedPermissions = permissions.map((p) => p.name).filter(isUnkeyPermission);

    return {
      id: editingKey.id,
      name: editingKey.name,
      permissions: validatedPermissions,
    };
  }, [editingKey, transport]);

  const columns = useMemo(
    () => createRootKeyColumns({ selectedRootKeyId, onEditKey: handleEditKey }),
    [selectedRootKeyId, handleEditKey],
  );

  return (
    <RootKeysTransportProvider transport={transport}>
      {variant === "resource-list" ? (
        <RootKeysResourceList
          rootKeys={rootKeys}
          isLoading={isInitialLoading}
          onSelect={handleRowClick}
          onEditKey={handleEditKey}
        />
      ) : (
        <DataTable
          data={rootKeys}
          columns={columns}
          getRowId={(rootKey) => rootKey.id}
          isLoading={isInitialLoading}
          onRowClick={handleRowClick}
          selectedItem={selectedRootKey}
          rowClassName={getRowClassNameMemoized}
          emptyState={
            <EmptyState frame="none">
              <EmptyStateHeader>
                <EmptyStateTitle>No Root Keys Found</EmptyStateTitle>
                <EmptyStateDescription>
                  There are no root keys configured yet. Create your first root key to start
                  managing permissions and access control.
                </EmptyStateDescription>
              </EmptyStateHeader>
              <EmptyStateActions>
                <a
                  href="https://www.unkey.com/docs/security/overview#root-keys"
                  target="_blank"
                  rel="noopener noreferrer"
                  className={buttonVariants({ variant: "outline", size: "md" })}
                >
                  <span className="flex items-center gap-2">
                    <IconBookBookmarkOutline18 />
                    Learn about Root Keys
                  </span>
                </a>
              </EmptyStateActions>
            </EmptyState>
          }
          config={TABLE_CONFIG}
          renderSkeletonRow={renderRootKeySkeletonRow}
          sorting={sorting}
          onSortingChange={onSortingChange}
        />
      )}
      <ListVariantDebugBar variant={variant} onChange={setVariant} />
      <PaginationFooter
        page={page}
        pageSize={pageSize}
        totalPages={totalPages}
        totalCount={totalCount}
        onPageChange={onPageChange}
        itemLabel="root keys"
        loading={isInitialLoading}
        disabled={isNavigating}
      />
      {editingKey &&
        (transport === "v2" ? (
          <V2RootKeyDialog
            rootKey={editingKey}
            isOpen={editDialogOpen}
            onOpenChange={handleEditDialogOpenChange}
          />
        ) : existingKey ? (
          <RootKeyDialog
            title="Edit root key"
            subTitle="Update the name and permissions for this root key"
            isOpen={editDialogOpen}
            onOpenChange={handleEditDialogOpenChange}
            editMode={true}
            existingKey={existingKey}
          />
        ) : null)}
    </RootKeysTransportProvider>
  );
};
