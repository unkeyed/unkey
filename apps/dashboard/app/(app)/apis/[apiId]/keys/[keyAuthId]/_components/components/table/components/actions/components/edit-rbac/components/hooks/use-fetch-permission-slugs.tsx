"use client";
import { trpc } from "@/lib/trpc/client";
import type { KeyPermission } from "@/lib/trpc/routers/key/rbac/connected-roles-and-perms";
import { useMemo } from "react";

export const useFetchPermissionSlugs = (
  roleIds: string[] = [],
  directPermissionIds: string[] = [], // Changed parameter name to be explicit
  allPermissions: KeyPermission[] = [],
  enabled = true,
) => {
  // Calculate all effective permissions: role-inherited + direct
  const allEffectivePermissionIds = useMemo(() => {
    // Get permissions from currently selected roles
    const rolePermissionIds = new Set<string>();
    allPermissions.forEach((permission) => {
      if (
        permission.source === "role" &&
        permission.roleId &&
        roleIds.includes(permission.roleId)
      ) {
        rolePermissionIds.add(permission.id);
      }
    });

    // Combine role permissions and direct permissions, removing duplicates
    const combined = new Set([...rolePermissionIds, ...directPermissionIds]);
    return Array.from(combined);
  }, [roleIds, directPermissionIds, allPermissions]);

  const { data, isLoading, error, refetch } = trpc.key.queryPermissionSlugs.useQuery(
    {
      roleIds,
      permissionIds: directPermissionIds,
    },
    {
      enabled,
    },
  );

  return {
    data,
    isLoading,
    error,
    refetch,
    hasData: !isLoading && data !== undefined,
    allEffectivePermissionIds, // Export this in case components need it
  };
};
