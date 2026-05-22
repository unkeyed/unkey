"use client";

import { Label } from "@/components/ui/label";
import { ScrollArea } from "@/components/ui/scroll-area";
import type { UnkeyPermission } from "@unkey/rbac";
import { Button } from "@unkey/ui";
import { FormInput } from "@unkey/ui";
import dynamic from "next/dynamic";
import { useMemo, useState } from "react";
import { createPortal } from "react-dom";
import { PermissionBadgeList } from "./components/permission-badge-list";
import { PermissionSheet } from "./components/permission-sheet";
import { ROOT_KEY_CONSTANTS, ROOT_KEY_MESSAGES } from "./constants";
import { useRootKeyDialog } from "./hooks/use-root-key-dialog";
import { WORKSPACE_SCOPE } from "./permissions";
import { RootKeySuccess } from "./root-key-success";

const DynamicDialogContainer = dynamic(
  () =>
    import("@unkey/ui")
      .then((mod) => ({
        default: mod.DialogContainer,
      }))
      .catch(() => ({
        default: () => <div role="alert">Failed to load dialog</div>,
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
  const [isSheetOpen, setIsSheetOpen] = useState(false);

  const handleOpenSheet = () => {
    setIsSheetOpen(true);
  };

  const handleDialogOpenChange = (open: boolean) => {
    // Suppress dialog close while the permission sheet is open — Esc/outside-close events
    // can otherwise propagate to the (non-modal) outer dialog and close both at once.
    if (!open && isSheetOpen) {
      setIsSheetOpen(false);
      return;
    }
    onOpenChange(open);
  };

  const {
    name,
    setName,
    selectedPermissions,
    allApis,
    apisLoading,
    allProjects,
    projectsLoading,
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
  } = useRootKeyDialog({
    isOpen,
    editMode,
    existingKey,
    onOpenChange,
  });

  const isMutating = key.isLoading || updateName.isLoading || updatePermissions.isLoading;
  const isBusy = isMutating || apisLoading || projectsLoading;

  const apiBadges = useMemo(
    () =>
      allApis.map((api) => ({
        id: api.id,
        name: api.name,
        scope: { kind: "api" as const, id: api.id, name: api.name },
      })),
    [allApis],
  );
  const projectBadges = useMemo(
    () =>
      allProjects.map((project) => ({
        id: project.id,
        name: project.name,
        scope: { kind: "project" as const, id: project.id, name: project.name },
      })),
    [allProjects],
  );

  const removePermission = (permission: UnkeyPermission) =>
    handlePermissionChange(selectedPermissions.filter((p) => p !== permission));

  const dialogContent = (
    <>
      <div className="flex flex-col p-6 gap-4">
        <div className="flex flex-col">
          <FormInput
            name="name"
            label={ROOT_KEY_MESSAGES.DESCRIPTIONS.KEY_NAME_LABEL}
            description={ROOT_KEY_MESSAGES.DESCRIPTIONS.KEY_NAME_DESCRIPTION}
            placeholder={ROOT_KEY_MESSAGES.PLACEHOLDERS.KEY_NAME}
            value={name}
            onChange={(e) => setName(e.target.value)}
          />
        </div>
        <div className="flex flex-col gap-2 mr-0">
          <Label className="text-[13px] font-regular text-gray-10">
            {ROOT_KEY_MESSAGES.DESCRIPTIONS.PERMISSIONS}
          </Label>
          <Button
            type="button"
            variant="outline"
            size="lg"
            className="rounded-lg font-light text-grayA-8 text-[13px] border border-gray-5 hover:border-gray-8 bg-gray-2 dark:bg-black focus-visible:outline-hidden focus-visible:ring-2 focus-visible:ring-gray-5 focus-visible:ring-offset-0"
            disabled={isBusy}
            onClick={handleOpenSheet}
          >
            {isBusy
              ? ROOT_KEY_MESSAGES.UI.LOADING
              : editMode
                ? ROOT_KEY_MESSAGES.UI.EDIT_PERMISSIONS
                : ROOT_KEY_MESSAGES.UI.SELECT_PERMISSIONS}
          </Button>
        </div>
      </div>
      <ScrollArea className="w-full overflow-y-auto pt-0 mb-4">
        <div className="flex flex-col px-6 py-0 gap-3">
          <PermissionBadgeList
            selectedPermissions={selectedPermissions}
            scope={WORKSPACE_SCOPE}
            title="Selected from"
            name="Workspace"
            expandCount={ROOT_KEY_CONSTANTS.EXPAND_COUNT}
            removePermission={removePermission}
          />
          {apiBadges.map((api) => (
            <PermissionBadgeList
              key={api.id}
              selectedPermissions={selectedPermissions}
              scope={api.scope}
              title="from"
              name={api.name}
              expandCount={ROOT_KEY_CONSTANTS.EXPAND_COUNT}
              removePermission={removePermission}
            />
          ))}
          {projectBadges.map((project) => (
            <PermissionBadgeList
              key={project.id}
              selectedPermissions={selectedPermissions}
              scope={project.scope}
              title="from project"
              name={project.name}
              expandCount={ROOT_KEY_CONSTANTS.EXPAND_COUNT}
              removePermission={removePermission}
            />
          ))}
        </div>
      </ScrollArea>
    </>
  );

  const footerContent = (
    <div className="w-full flex flex-col gap-2 items-center justify-center">
      <Button
        variant="primary"
        size="xlg"
        className="w-full rounded-lg"
        disabled={!hasChanges || isMutating || !selectedPermissions.length}
        onClick={handleCreateKey}
        loading={isBusy}
      >
        {editMode ? ROOT_KEY_MESSAGES.UI.UPDATE_ROOT_KEY : ROOT_KEY_MESSAGES.UI.CREATE_ROOT_KEY}
      </Button>
      <div className="text-gray-9 text-xs">
        {editMode
          ? ROOT_KEY_MESSAGES.DESCRIPTIONS.IMMEDIATE_UPDATE
          : ROOT_KEY_MESSAGES.DESCRIPTIONS.IMMEDIATE_CREATE}
      </div>
    </div>
  );

  return (
    <>
      {/* The outer dialog is non-modal so its react-remove-scroll doesn't block wheel events
          inside the sheet (which is portaled outside the dialog's DOM subtree). We render our
          own backdrop here to replace the modal overlay we lost. */}
      {!key.data?.key &&
        isOpen &&
        typeof document !== "undefined" &&
        createPortal(
          // @dh hacky fix, delete asap
          // biome-ignore lint/a11y/useKeyWithClickEvents: decorative backdrop (aria-hidden); keyboard close is handled by Radix Esc handling on the dialog.
          <div
            className="fixed inset-0 z-40 bg-black/30 backdrop-blur-xs"
            aria-hidden="true"
            onClick={() => handleDialogOpenChange(false)}
          />,
          document.body,
        )}
      {!key.data?.key && (
        <DynamicDialogContainer
          isOpen={isOpen}
          onOpenChange={handleDialogOpenChange}
          title={title}
          contentClassName="p-0 mb-0 gap-0"
          className="max-w-[460px]"
          subTitle={subTitle}
          footer={footerContent}
          modal={false}
          preventOutsideClose={isSheetOpen}
        >
          {dialogContent}
        </DynamicDialogContainer>
      )}
      <PermissionSheet
        selectedPermissions={selectedPermissions}
        apis={allApis}
        projects={allProjects}
        onChange={handlePermissionChange}
        loadMore={fetchMoreApis}
        hasNextPage={hasNextPage}
        isFetchingNextPage={isFetchingNextPage}
        editMode={editMode}
        open={isSheetOpen}
        onOpenChange={setIsSheetOpen}
      />
      <RootKeySuccess keyValue={key.data?.key} onClose={handleClose} />
    </>
  );
};
