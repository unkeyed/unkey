"use client";

import { Label } from "@/components/ui/label";
import { ScrollArea } from "@/components/ui/scroll-area";
import type { UnkeyPermission } from "@unkey/rbac";
import { Button, FormInput } from "@unkey/ui";
import dynamic from "next/dynamic";
import { PermissionBadgeList } from "./components/permission-badge-list";
import { PermissionSheet } from "./components/permission-sheet";
import { ROOT_KEY_CONSTANTS, ROOT_KEY_MESSAGES } from "./constants";
import { useRootKeyDialog } from "./hooks/use-root-key-dialog";
import { RootKeySuccess } from "./root-key-success";

const DynamicDialogContainer = dynamic(
  () =>
    import("@unkey/ui").then((mod) => ({
      default: mod.DialogContainer,
    })),
  { ssr: false },
);

type RootKeyDialogProps = {
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
}: RootKeyDialogProps) => {
  const {
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
  } = useRootKeyDialog({
    editMode,
    existingKey,
    onOpenChange,
  });

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
              {editMode
                ? ROOT_KEY_MESSAGES.UI.UPDATE_ROOT_KEY
                : ROOT_KEY_MESSAGES.UI.CREATE_ROOT_KEY}
            </Button>
            <div className="text-gray-9 text-xs">
              {editMode
                ? ROOT_KEY_MESSAGES.DESCRIPTIONS.IMMEDIATE_UPDATE
                : ROOT_KEY_MESSAGES.DESCRIPTIONS.IMMEDIATE_CREATE}
            </div>
          </div>
        }
      >
        <div className="flex flex-col p-6 gap-4">
          <div className="flex flex-col">
            <FormInput
              name="name"
              label="Name"
              description={ROOT_KEY_MESSAGES.DESCRIPTIONS.KEY_NAME}
              placeholder={ROOT_KEY_MESSAGES.PLACEHOLDERS.KEY_NAME}
              value={name}
              onChange={(e) => setName(e.target.value)}
            />
          </div>
          <div className="flex flex-col gap-2">
            <Label className="text-[13px] font-regular text-gray-10">
              {ROOT_KEY_MESSAGES.DESCRIPTIONS.PERMISSIONS}
            </Label>
            <PermissionSheet
              selectedPermissions={selectedPermissions}
              apis={allApis}
              onChange={handlePermissionChange}
              loadMore={fetchMoreApis}
              hasNextPage={hasNextPage}
              isFetchingNextPage={isFetchingNextPage}
              editMode={editMode}
            >
              <Button
                type="button"
                variant="outline"
                size="md"
                className="w-fit rounded-lg pl-3"
                disabled={apisLoading && !editMode && selectedPermissions.length === 0}
              >
                {apisLoading && !editMode && selectedPermissions.length === 0
                  ? ROOT_KEY_MESSAGES.UI.LOADING
                  : ROOT_KEY_MESSAGES.UI.SELECT_PERMISSIONS}
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
              expandCount={ROOT_KEY_CONSTANTS.EXPAND_COUNT}
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
                expandCount={ROOT_KEY_CONSTANTS.EXPAND_COUNT}
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
