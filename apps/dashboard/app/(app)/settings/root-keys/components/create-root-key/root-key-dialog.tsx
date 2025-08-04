"use client";
import { Label } from "@/components/ui/label";
import { ScrollArea } from "@/components/ui/scroll-area";
import { trpc } from "@/lib/trpc/client";
import type { UnkeyPermission } from "@unkey/rbac";
import { Button, FormInput, toast } from "@unkey/ui";
import dynamic from "next/dynamic";
import { useCallback, useEffect, useMemo, useState } from "react";
import { PermissionBadgeList } from "./components/permission-badge-list";
import { PermissionSheet } from "./components/permission-sheet";
import { RootKeySuccess } from "./root-key-success";

const DynamicDialogContainer = dynamic(
  () =>
    import("@unkey/ui").then((mod) => ({
      default: mod.DialogContainer,
    })),
  { ssr: false },
);

const DEFAULT_LIMIT = 10;

type Props = {
  title: string;
  subTitle: string;
  isOpen: boolean;
  onOpenChange: (open: boolean) => void;
  editMode?: boolean;
  existingKey?: {
    id: string;
    name: string | null;
    permissions: UnkeyPermission[];
  };
};

export const RootKeyDialog = ({
  title,
  subTitle,
  isOpen,
  onOpenChange,
  editMode = false,
  existingKey,
}: Props) => {
  const trpcUtils = trpc.useUtils();
  const [name, setName] = useState(existingKey?.name ?? "");
  const [selectedPermissions, setSelectedPermissions] = useState<UnkeyPermission[]>(
    existingKey?.permissions ?? [],
  );

  const {
    data: apisData,
    fetchNextPage,
    hasNextPage,
    isFetchingNextPage,
    isLoading: apisLoading,
  } = trpc.api.overview.query.useInfiniteQuery(
    { limit: DEFAULT_LIMIT },
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

  function fetchMoreApis() {
    if (hasNextPage) {
      fetchNextPage();
    }
  }

  const handlePermissionChange = useCallback(
    (permissions: UnkeyPermission[]) => {
      // Only update if APIs are loaded or we're in edit mode with existing permissions
      if (!apisLoading && (apisData?.pages || (editMode && existingKey?.permissions))) {
        setSelectedPermissions(permissions);
      }
    },
    [apisLoading, apisData?.pages, editMode, existingKey?.permissions],
  );

  const handleCreateKey = async () => {
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

      toast.success("Root key updated successfully!");
      onOpenChange(false);
    } else {
      // Create new key
      key.mutate({
        name: name && name.length > 0 ? name : undefined,
        permissions: selectedPermissions,
      });
    }
  };

  const handleClose = () => {
    onOpenChange(false);
    key.reset();
    setSelectedPermissions([]);
    setName("");
  };

  // Reset form when dialog opens/closes or when existingKey changes
  useEffect(() => {
    if (isOpen && existingKey?.permissions) {
      setName(existingKey.name ?? "");
      setSelectedPermissions(existingKey.permissions);
    } else if (!isOpen) {
      setName("");
      setSelectedPermissions([]);
    }
  }, [isOpen, existingKey]);

  return (
    <>
      <DynamicDialogContainer
        isOpen={isOpen}
        onOpenChange={onOpenChange}
        title={title}
        contentClassName="p-0 mb-0 gap-0"
        className="max-w-[460px]"
        subTitle={subTitle}
        footer={
          <div className="w-full flex flex-col gap-2 items-center justify-center">
            <Button
              variant="primary"
              size="xlg"
              className="w-full rounded-lg"
              disabled={selectedPermissions.length === 0}
              onClick={handleCreateKey}
              loading={key.isLoading || updateName.isLoading || updatePermissions.isLoading}
            >
              {editMode ? "Update root key" : "Create root key"}
            </Button>
            <div className="text-gray-9 text-xs">
              {editMode
                ? "This root key will be updated immediately"
                : "This root key will be created immediately"}
            </div>
          </div>
        }
      >
        <div className="flex flex-col p-6 gap-4">
          <div className="flex flex-col">
            <FormInput
              name="name"
              label="Name"
              description="Give your key a name, this is not customer facing."
              placeholder="e.g. Vercel Production"
              value={name}
              onChange={(e) => setName(e.target.value)}
            />
          </div>
          <div className="flex flex-col gap-2">
            <Label className="text-[13px] font-regular text-gray-10">Permissions</Label>
            <PermissionSheet
              selectedPermissions={selectedPermissions}
              apis={allApis}
              onChange={handlePermissionChange}
              loadMore={fetchMoreApis}
              hasNextPage={hasNextPage}
              isFetchingNextPage={isFetchingNextPage}
              editMode={editMode}
              isLoading={apisLoading}
            >
              <Button
                type="button"
                variant="outline"
                size="md"
                className="w-fit rounded-lg pl-3"
                disabled={apisLoading && !editMode && selectedPermissions.length === 0}
              >
                {apisLoading && !editMode && selectedPermissions.length === 0
                  ? "Loading..."
                  : "Select Permissions..."}
              </Button>
            </PermissionSheet>
          </div>
        </div>
        <ScrollArea className="w-full overflow-y-auto pt-0 mb-4">
          <div className="flex flex-col px-6 py-0 gap-3">
            <PermissionBadgeList
              selectedPermissions={selectedPermissions}
              apiId={"workspace"}
              title="Selected from"
              name="Workspace"
              expandCount={3}
              removePermission={(permission) =>
                handlePermissionChange(selectedPermissions.filter((p) => p !== permission))
              }
            />
            {allApis.map((api) => (
              <PermissionBadgeList
                key={api.id}
                selectedPermissions={selectedPermissions}
                apiId={api.id}
                title="from"
                name={api.name}
                expandCount={3}
                removePermission={(permission) =>
                  handlePermissionChange(selectedPermissions.filter((p) => p !== permission))
                }
              />
            ))}
          </div>
        </ScrollArea>
      </DynamicDialogContainer>
      <RootKeySuccess
        keyValue={key.data?.key}
        keyId={key.data?.keyId}
        name={name}
        onClose={handleClose}
      />
    </>
  );
};
