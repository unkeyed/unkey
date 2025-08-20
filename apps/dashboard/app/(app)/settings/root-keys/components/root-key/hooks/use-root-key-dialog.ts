import { trpc } from "@/lib/trpc/client";
import type { UnkeyPermission } from "@unkey/rbac";
import { toast } from "@unkey/ui";
import { useCallback, useEffect, useMemo, useState } from "react";
import { ROOT_KEY_CONSTANTS, ROOT_KEY_MESSAGES } from "../constants";

// Utility function for robust permission array comparison
function arePermissionArraysEqual(
  permissions1: UnkeyPermission[],
  permissions2: UnkeyPermission[],
): boolean {
  // Handle edge cases
  if (permissions1 === permissions2) {
    return true;
  }
  if (permissions1.length !== permissions2.length) {
    return false;
  }
  if (permissions1.length === 0) {
    return true;
  }

  // Normalize permissions by sorting and creating a canonical representation
  const normalizePermissions = (perms: UnkeyPermission[]): string[] => {
    return perms
      .map((p) => String(p)) // Ensure all permissions are strings
      .sort(); // Sort for consistent comparison
  };

  const normalized1 = normalizePermissions(permissions1);
  const normalized2 = normalizePermissions(permissions2);

  // Compare each permission
  for (let i = 0; i < normalized1.length; i++) {
    if (normalized1[i] !== normalized2[i]) {
      return false;
    }
  }

  return true;
}

type UseRootKeyDialogProps = {
  editMode?: boolean;
  existingKey?: {
    id: string;
    name: string | null;
    permissions: UnkeyPermission[];
  };
  onOpenChange: (open: boolean) => void;
};

export function useRootKeyDialog({
  editMode = false,
  existingKey,
  onOpenChange,
}: UseRootKeyDialogProps) {
  const trpcUtils = trpc.useUtils();
  const [name, setName] = useState(existingKey?.name ?? "");
  const [selectedPermissions, setSelectedPermissions] = useState<UnkeyPermission[]>(
    existingKey?.permissions ?? [],
  );

  // Fetch APIs
  const {
    data: apisData,
    fetchNextPage,
    hasNextPage,
    isFetchingNextPage,
    isLoading: apisLoading,
  } = trpc.api.overview.query.useInfiniteQuery(
    { limit: ROOT_KEY_CONSTANTS.DEFAULT_LIMIT },
    {
      getNextPageParam: (lastPage) => lastPage.nextCursor,
    },
  );

  const allApis = useMemo(() => {
    if (!apisData?.pages) {
      return [];
    }
    return apisData.pages.flatMap((page) => {
      return page.apiList.map((api) => ({
        id: api.id,
        name: api.name,
      }));
    });
  }, [apisData]);

  // Mutations
  const key = trpc.rootKey.create.useMutation({
    onSuccess() {
      toast.success(ROOT_KEY_MESSAGES.SUCCESS.ROOT_KEY_GENERATED);
      trpcUtils.settings.rootKeys.query.invalidate();
    },
    onError(err: { message: string }) {
      toast.error(err.message);
    },
  });

  const updateName = trpc.rootKey.update.name.useMutation({
    onSuccess() {
      toast.success(ROOT_KEY_MESSAGES.SUCCESS.ROOT_KEY_UPDATED_NAME);
      trpcUtils.settings.rootKeys.query.invalidate();
    },
    onError(err: { message: string }) {
      toast.error(err.message);
    },
  });

  const updatePermissions = trpc.rootKey.update.permissions.useMutation({
    onSuccess() {
      const count = selectedPermissions.length;
      toast.success(`${ROOT_KEY_MESSAGES.SUCCESS.ROOT_KEY_UPDATED_PERMISSIONS} ${count}`);
      trpcUtils.settings.rootKeys.query.invalidate();
    },
    onError(err: { message: string }) {
      toast.error(err.message);
    },
  });

  const fetchMoreApis = useCallback(() => {
    if (hasNextPage) {
      fetchNextPage();
    }
  }, [hasNextPage, fetchNextPage]);

  const handlePermissionChange = useCallback(
    (permissions: UnkeyPermission[]) => {
      // Prevent updates while APIs are loading in create mode
      const canUpdate = !apisLoading || editMode;
      if (canUpdate) {
        setSelectedPermissions(permissions);
      }
    },
    [apisLoading, editMode],
  );

  const handleCreateKey = useCallback(async () => {
    if (editMode && existingKey) {
      // Update existing key name if changed
      const nameChanged = name !== existingKey.name && name !== "" && name !== null;

      if (nameChanged) {
        await updateName.mutateAsync({
          keyId: existingKey.id,
          name: name && name.length > 0 ? name : null,
        });
      }
      if (
        selectedPermissions.length > 0 &&
        !arePermissionArraysEqual(selectedPermissions, existingKey.permissions)
      ) {
        // Update permissions using the new bulk update route
        await updatePermissions.mutateAsync({
          keyId: existingKey.id,
          permissions: selectedPermissions,
        });
      }

      onOpenChange(false);
    } else {
      // Create new key
      key.mutate({
        name: name && name.length > 0 ? name : undefined,
        permissions: selectedPermissions,
      });
    }
  }, [
    editMode,
    existingKey,
    name,
    selectedPermissions,
    updateName,
    updatePermissions,
    key,
    onOpenChange,
  ]);

  const handleClose = useCallback(() => {
    onOpenChange(false);
    key.reset();
    setSelectedPermissions([]);
    setName("");
  }, [onOpenChange, key]);

  // Reset form when dialog opens/closes or when existingKey changes
  useEffect(() => {
    if (existingKey?.permissions) {
      setName(existingKey.name ?? "");
      setSelectedPermissions(existingKey.permissions);
    } else {
      setName("");
      setSelectedPermissions([]);
    }
  }, [existingKey]);

  // Check if there are any changes to enable/disable the update button
  const hasChanges = useMemo(() => {
    if (!editMode || !existingKey) {
      // For create mode, button should be enabled if there are permissions
      return selectedPermissions.length > 0;
    }

    // For edit mode, check if name or permissions have changed
    const nameChanged = name !== existingKey.name;
    const permissionsChanged = !arePermissionArraysEqual(
      selectedPermissions,
      existingKey.permissions,
    );

    return nameChanged || permissionsChanged;
  }, [editMode, existingKey, name, selectedPermissions]);

  return {
    name,
    setName,
    selectedPermissions,
    allApis,
    apisLoading,
    hasNextPage,
    isFetchingNextPage,
    key,
    updateName,
    updatePermissions,
    fetchMoreApis,
    handlePermissionChange,
    handleCreateKey,
    handleClose,
    hasChanges,
  };
}
