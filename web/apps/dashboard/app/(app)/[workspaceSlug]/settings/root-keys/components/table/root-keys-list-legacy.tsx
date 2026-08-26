"use client";

import type { RootKey } from "@/lib/trpc/routers/settings/root-keys/query";
import type { UnkeyPermission } from "@unkey/rbac";
import { unkeyPermissionValidation } from "@unkey/rbac";
import { useCallback, useMemo, useState } from "react";
import { RootKeyDialog } from "../dialog/root-key-dialog";
import { RootKeysDataTable } from "./root-keys-data-table";

// Type guard function to check if a string is a valid UnkeyPermission
const isUnkeyPermission = (permissionName: string): permissionName is UnkeyPermission => {
  const result = unkeyPermissionValidation.safeParse(permissionName);
  return result.success;
};

export function RootKeysListLegacy() {
  const [editingKey, setEditingKey] = useState<RootKey | null>(null);
  const [isOpen, setIsOpen] = useState(false);

  const edit = useCallback((rootKey: RootKey) => {
    setEditingKey(rootKey);
    setIsOpen(true);
  }, []);

  const existingKey = useMemo(() => {
    if (!editingKey) {
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
  }, [editingKey]);

  return (
    <>
      <RootKeysDataTable selectedKeyId={editingKey?.id ?? null} onEditKey={edit} />
      {editingKey && existingKey && (
        <RootKeyDialog
          title="Edit Root Key"
          subTitle="Update the name and permissions for this Root Key"
          isOpen={isOpen}
          onOpenChange={(open) => {
            setIsOpen(open);
            if (!open) {
              setEditingKey(null);
            }
          }}
          editMode={true}
          existingKey={existingKey}
        />
      )}
    </>
  );
}
