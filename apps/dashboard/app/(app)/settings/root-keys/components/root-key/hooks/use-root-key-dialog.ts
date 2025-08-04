import { trpc } from "@/lib/trpc/client";
import type { UnkeyPermission } from "@unkey/rbac";
import { toast } from "@unkey/ui";
import { useCallback, useEffect, useMemo, useState } from "react";
import { ROOT_KEY_CONSTANTS, ROOT_KEY_MESSAGES } from "../constants";

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
      trpcUtils.settings.rootKeys.query.invalidate();
    },
    onError(err: { message: string }) {
      console.error(err);
      toast.error(err.message);
    },
  });

  const updateName = trpc.rootKey.update.name.useMutation({
    onSuccess() {
      trpcUtils.settings.rootKeys.query.invalidate();
    },
    onError(err: { message: string }) {
      console.error(err);
      toast.error(err.message);
    },
  });

  const updatePermissions = trpc.rootKey.update.permissions.useMutation({
    onSuccess() {
      trpcUtils.settings.rootKeys.query.invalidate();
    },
    onError(err: { message: string }) {
      console.error("Permission update error:", err);
      console.error("Selected permissions:", selectedPermissions);
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
      // Only update if APIs are loaded or we're in edit mode with existing permissions
      if (!apisLoading && (apisData?.pages || (editMode && existingKey?.permissions))) {
        setSelectedPermissions(permissions);
      }
    },
    [apisLoading, apisData?.pages, editMode, existingKey?.permissions],
  );

  const handleCreateKey = useCallback(async () => {
    if (editMode && existingKey) {
      // Update existing key name if changed
      const nameChanged = name !== existingKey.name;
      if (nameChanged) {
        await updateName.mutateAsync({
          keyId: existingKey.id,
          name: name && name.length > 0 ? name : null,
        });
      }

      // Update permissions using the new bulk update route
      await updatePermissions.mutateAsync({
        keyId: existingKey.id,
        permissions: selectedPermissions,
      });

      toast.success(ROOT_KEY_MESSAGES.SUCCESS.ROOT_KEY_UPDATED);
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
  };
}
