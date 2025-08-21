import type { UnkeyPermission } from "@unkey/rbac";
import { useCallback, useMemo, useState } from "react";
import type { ChangeEvent } from "react";
import { apiPermissions, workspacePermissions } from "../permissions";
import { hasPermissionResults } from "../utils/permissions";

type UsePermissionSheetProps = {
  apis: { id: string; name: string }[];
  selectedPermissions: UnkeyPermission[];
  onChange?: (permissions: UnkeyPermission[]) => void;
  editMode?: boolean;
};

export function usePermissionSheet({
  apis,
  selectedPermissions,
  onChange,
}: UsePermissionSheetProps) {
  const [searchValue, setSearchValue] = useState<string | undefined>(undefined);
  const [isProcessing, setIsProcessing] = useState(false);

  // Memoize workspace permissions Set to avoid duplication and GC churn
  const workspacePermsSet = useMemo(
    () =>
      new Set<UnkeyPermission>(
        Object.values(workspacePermissions).flatMap((category) =>
          Object.values(category).map((item) => item.permission),
        ),
      ),
    [],
  );

  const handleSearchChange = useCallback((e: ChangeEvent<HTMLInputElement>) => {
    setIsProcessing(true);
    if (e.target.value === "") {
      setSearchValue(undefined);
    } else {
      setSearchValue(e.target.value);
    }
    setIsProcessing(false);
  }, []);

  const handleApiPermissionChange = useCallback(
    (apiId: string, permissions: UnkeyPermission[]) => {
      if (!onChange) {
        return;
      }

      const workspacePerms = selectedPermissions.filter((permission) =>
        workspacePermsSet.has(permission),
      );

      // Get other APIs' permissions (exclude current API)
      const otherApisPermsSet = new Set<UnkeyPermission>();
      for (const api of apis) {
        if (api.id === apiId) {
          continue;
        }
        for (const perm of Object.values(apiPermissions(api.id)).flatMap((category) =>
          Object.values(category).map((item) => item.permission),
        )) {
          otherApisPermsSet.add(perm);
        }
      }
      const otherApiPerms = selectedPermissions.filter(
        (permission) => !workspacePermsSet.has(permission) && otherApisPermsSet.has(permission),
      );

      // Combine all permissions (de-duplicated)
      const allPermissions = Array.from(
        new Set<UnkeyPermission>([...workspacePerms, ...otherApiPerms, ...permissions]),
      );
      onChange(allPermissions);
    },
    [selectedPermissions, apis, onChange, workspacePermsSet],
  );

  const handleWorkspacePermissionChange = useCallback(
    (permissions: UnkeyPermission[]) => {
      if (!onChange) {
        return;
      }

      const apiPerms = selectedPermissions.filter(
        (permission) => !workspacePermsSet.has(permission),
      );

      // Combine workspace and API permissions (de-duplicated)
      const allPermissions = Array.from(new Set<UnkeyPermission>([...permissions, ...apiPerms]));
      onChange(allPermissions);
    },
    [selectedPermissions, onChange, workspacePermsSet],
  );

  // Check if all permission lists are empty after filtering
  const hasNoResults = useMemo(() => {
    // Check workspace permissions
    const workspaceHasResults = hasPermissionResults(workspacePermissions, searchValue);

    // Check API permissions
    const anyApiHasResults = apis.some((api) => {
      const apiPerms = apiPermissions(api.id);
      return hasPermissionResults(apiPerms, searchValue);
    });

    return !workspaceHasResults && !anyApiHasResults;
  }, [searchValue, apis]);

  return {
    searchValue,
    isProcessing,
    hasNoResults,
    handleSearchChange,
    handleApiPermissionChange,
    handleWorkspacePermissionChange,
  };
}
