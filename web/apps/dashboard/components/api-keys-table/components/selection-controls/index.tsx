import { RECENTLY_USED_WINDOW_LABEL, isRecentlyUsed } from "@/lib/recently-used-key";
import type { KeyDetails } from "@/lib/trpc/routers/api/keys/query-api-keys/schema";
import {
  IconArrowsOppositeDirectionYOutline18,
  IconBanOutline18,
  IconCircleCheckOutline18,
  IconTrashOutline18,
  IconTriangleWarningOutline18,
} from "@unkey/icons";
import { ConfirmPopover, SelectionBar, SelectionBarAction } from "@unkey/ui";
import { useEffect, useRef, useState } from "react";
import { useDeleteKey } from "../actions/components/hooks/use-delete-key";
import { useBatchUpdateKeyStatus } from "../actions/components/hooks/use-update-key-status";
import { BatchEditExternalId } from "./components/batch-edit-external-id";

type SelectionControlsProps = {
  selectedKeys: Set<string>;
  setSelectedKeys: (keys: Set<string>) => void;
  keys: KeyDetails[];
  getSelectedKeysState: () => "all-enabled" | "all-disabled" | "mixed" | null;
};

export const SelectionControls = ({
  selectedKeys,
  keys,
  setSelectedKeys,
  getSelectedKeysState,
}: SelectionControlsProps) => {
  const [isBatchEditExternalIdOpen, setIsBatchEditExternalIdOpen] = useState(false);
  const [isDisableConfirmOpen, setIsDisableConfirmOpen] = useState(false);
  const [isDeleteConfirmOpen, setIsDeleteConfirmOpen] = useState(false);

  const disableButtonRef = useRef<HTMLButtonElement>(null);
  const deleteButtonRef = useRef<HTMLButtonElement>(null);

  useEffect(() => {
    if (selectedKeys.size === 0) {
      setIsDisableConfirmOpen(false);
      setIsDeleteConfirmOpen(false);
      setIsBatchEditExternalIdOpen(false);
    }
  }, [selectedKeys.size]);

  const updateKeyStatus = useBatchUpdateKeyStatus();
  const deleteKey = useDeleteKey(() => {
    setSelectedKeys(new Set());
  });

  const handleDisableButtonClick = () => {
    setIsDisableConfirmOpen(true);
  };

  const performDisableKeys = () => {
    const keyIds = Array.from(selectedKeys);
    if (keyIds.length === 0) {
      return;
    }
    updateKeyStatus.mutate({
      enabled: false,
      keyIds,
    });
  };

  const handleDeleteButtonClick = () => {
    setIsDeleteConfirmOpen(true);
  };

  const performKeyDeletion = () => {
    const keyIds = Array.from(selectedKeys);
    if (keyIds.length === 0) {
      return;
    }
    deleteKey.mutate({
      keyIds,
    });
  };

  const keysWithExternalIds = keys.filter(
    (key) => selectedKeys.has(key.id) && key.identity_id,
  ).length;

  const recentlyUsedCount = keys.filter(
    (key) => selectedKeys.has(key.id) && isRecentlyUsed(key.last_used_at),
  ).length;

  const recentlyUsedSubject =
    selectedKeys.size === 1
      ? "This key was"
      : `${recentlyUsedCount} of the selected keys ${recentlyUsedCount > 1 ? "were" : "was"}`;
  const recentlyUsedSummary = `${recentlyUsedSubject} used in the last ${RECENTLY_USED_WINDOW_LABEL}`;
  const recentlyUsedNotice =
    recentlyUsedCount > 0 ? `${recentlyUsedSummary} and may still be live. ` : "";

  return (
    <>
      <SelectionBar
        count={selectedKeys.size}
        onClear={() => setSelectedKeys(new Set())}
        detail={
          recentlyUsedCount > 0 && (
            <div className="flex min-w-0 max-w-72 items-center @max-3xl:hidden gap-1.5 text-sm text-warning-11">
              <IconTriangleWarningOutline18 className="size-3.5 shrink-0" />
              <span className="truncate">{recentlyUsedSummary}</span>
            </div>
          )
        }
      >
        <SelectionBarAction
          onClick={() => setIsBatchEditExternalIdOpen(true)}
          icon={<IconArrowsOppositeDirectionYOutline18 />}
          label="Change External ID"
        />
        <SelectionBarAction
          disabled={getSelectedKeysState() !== "all-disabled" || updateKeyStatus.isLoading}
          loading={updateKeyStatus.isLoading}
          onClick={() => {
            const keyIds = Array.from(selectedKeys);
            if (keyIds.length === 0) {
              return;
            }
            updateKeyStatus.mutate({
              enabled: true,
              keyIds,
            });
          }}
          icon={<IconCircleCheckOutline18 />}
          label="Enable key"
        />
        <SelectionBarAction
          disabled={getSelectedKeysState() !== "all-enabled" || updateKeyStatus.isLoading}
          loading={updateKeyStatus.isLoading}
          onClick={handleDisableButtonClick}
          ref={disableButtonRef}
          icon={<IconBanOutline18 />}
          label="Disable key"
        />
        <SelectionBarAction
          disabled={deleteKey.isLoading}
          loading={deleteKey.isLoading}
          onClick={handleDeleteButtonClick}
          ref={deleteButtonRef}
          icon={<IconTrashOutline18 />}
          label="Delete key"
        />
      </SelectionBar>

      <ConfirmPopover
        isOpen={isDisableConfirmOpen}
        onOpenChange={setIsDisableConfirmOpen}
        onConfirm={performDisableKeys}
        triggerRef={disableButtonRef}
        title="Confirm disabling keys"
        description={`This will disable ${selectedKeys.size} key${
          selectedKeys.size > 1 ? "s" : ""
        } and prevent any verification requests from being processed.`}
        confirmButtonText="Disable keys"
        cancelButtonText="Cancel"
        variant="danger"
      />

      <ConfirmPopover
        isOpen={isDeleteConfirmOpen}
        onOpenChange={setIsDeleteConfirmOpen}
        onConfirm={performKeyDeletion}
        triggerRef={deleteButtonRef}
        title="Confirm key deletion"
        description={`${recentlyUsedNotice}This action is irreversible. All data associated with ${
          selectedKeys.size > 1 ? "these keys" : "this key"
        } will be permanently deleted.`}
        confirmButtonText={`Delete key${selectedKeys.size > 1 ? "s" : ""}`}
        cancelButtonText="Cancel"
        variant="danger"
      />

      {isBatchEditExternalIdOpen && (
        <BatchEditExternalId
          selectedKeyIds={Array.from(selectedKeys)}
          keysWithExternalIds={keysWithExternalIds}
          isOpen={isBatchEditExternalIdOpen}
          onClose={() => setIsBatchEditExternalIdOpen(false)}
        />
      )}
    </>
  );
};
