import type { UnkeyPermission } from "@unkey/rbac";
import { useCallback, useMemo, useState } from "react";
import { apiPermissions, workspacePermissions } from "../../../[keyId]/permissions/permissions";
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

  const handleSearchChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
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

      // Get workspace permissions
      const workspacePerms = selectedPermissions.filter((permission) => {
        const workspacePermsList = Object.values(workspacePermissions).flatMap((category) =>
          Object.values(category).map((item) => item.permission),
        );
        return workspacePermsList.includes(permission);
      });

      // Get other API permissions
      const otherApiPerms = selectedPermissions.filter((permission) => {
        const workspacePermsList = Object.values(workspacePermissions).flatMap((category) =>
          Object.values(category).map((item) => item.permission),
        );
        if (workspacePermsList.includes(permission)) {
          return false;
        }

        for (const api of apis) {
          if (api.id === apiId) {
            continue;
          }
          const apiPermsList = Object.values(apiPermissions(api.id)).flatMap((category) =>
            Object.values(category).map((item) => item.permission),
          );
          if (apiPermsList.includes(permission)) {
            return true;
          }
        }
        return false;
      });

      // Combine all permissions
      const allPermissions = [...workspacePerms, ...otherApiPerms, ...permissions];
      onChange(allPermissions);
    },
    [selectedPermissions, apis, onChange],
  );

  const handleWorkspacePermissionChange = useCallback(
    (permissions: UnkeyPermission[]) => {
      if (!onChange) {
        return;
      }

      // Get all current API permissions
      const apiPerms = selectedPermissions.filter((permission) => {
        const workspacePermsList = Object.values(workspacePermissions).flatMap((category) =>
          Object.values(category).map((item) => item.permission),
        );
        return !workspacePermsList.includes(permission);
      });

      // Combine workspace and API permissions
      const allPermissions = [...permissions, ...apiPerms];
      onChange(allPermissions);
    },
    [selectedPermissions, onChange],
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

    return !workspaceHasResults && (apis.length === 0 || !anyApiHasResults);
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
