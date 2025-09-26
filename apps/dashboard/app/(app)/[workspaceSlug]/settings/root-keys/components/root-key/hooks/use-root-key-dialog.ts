import { trpc } from "@/lib/trpc/client";
import type { UnkeyPermission } from "@unkey/rbac";
import { toast } from "@unkey/ui";
import { useCallback, useEffect, useMemo, useState } from "react";
import { ROOT_KEY_CONSTANTS, ROOT_KEY_MESSAGES } from "../constants";

// Utility function for robust permission array comparison using Set-based equality check
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

  // Convert first array to Set for O(1) lookups, ensuring values are strings
  const permissionSet = new Set(permissions1.map((p) => String(p)));

  // Check if every permission in the second array exists in the Set
  for (const permission of permissions2) {
    if (!permissionSet.has(String(permission))) {
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
    onError(err) {
      if (err.data?.code === "BAD_REQUEST") {
        toast.error("You need to add at least one permission.");
      } else {
        toast.error("Something went wrong. Please try again.");
      }
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
    onSuccess(_data, variables) {
      const count = variables?.permissions?.length ?? 0;
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
      // Normalize name and update if changed (allow clearing to null)
      const normalizedName = name.trim();
      const nameChanged = normalizedName !== (existingKey.name ?? "");

      if (nameChanged) {
        const nameToSend = normalizedName === "" ? null : normalizedName;
        await updateName.mutateAsync({
          keyId: existingKey.id,
          name: nameToSend,
        });
      }

      if (!arePermissionArraysEqual(selectedPermissions, existingKey.permissions)) {
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
