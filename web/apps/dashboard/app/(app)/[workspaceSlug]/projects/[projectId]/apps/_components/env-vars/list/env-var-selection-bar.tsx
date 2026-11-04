"use client";

import type { EnvVar } from "@/lib/collections/deploy/env-vars";
import { plural } from "@/lib/fmt";
import { IconEyeSlashOutline18, IconTrashOutline18 } from "@unkey/icons";
import { ConfirmPopover, SelectionBar, SelectionBarAction } from "@unkey/ui";
import { useRef, useState } from "react";
import { useBulkEnvVarActions } from "./use-bulk-env-var-actions";

export function EnvVarSelectionBar({
  selected,
  onClearSelection,
}: {
  selected: EnvVar[];
  onClearSelection: () => void;
}) {
  const { deleteSelected, makeSelectedSensitive, isMakingSensitive } = useBulkEnvVarActions(
    selected,
    onClearSelection,
  );
  const selectedCount = selected.length;
  const [isDeleteConfirmOpen, setIsDeleteConfirmOpen] = useState(false);
  const [isSensitiveConfirmOpen, setIsSensitiveConfirmOpen] = useState(false);
  const deleteButtonRef = useRef<HTMLButtonElement>(null);
  const sensitiveButtonRef = useRef<HTMLButtonElement>(null);

  return (
    <>
      <SelectionBar count={selectedCount} onClear={onClearSelection}>
        <SelectionBarAction
          onClick={() => setIsSensitiveConfirmOpen(true)}
          ref={sensitiveButtonRef}
          loading={isMakingSensitive}
          disabled={isMakingSensitive}
          icon={<IconEyeSlashOutline18 />}
          label="Make Sensitive"
        />
        <SelectionBarAction
          onClick={() => setIsDeleteConfirmOpen(true)}
          ref={deleteButtonRef}
          icon={<IconTrashOutline18 />}
          label="Delete"
        />
      </SelectionBar>

      <ConfirmPopover
        isOpen={isSensitiveConfirmOpen}
        onOpenChange={setIsSensitiveConfirmOpen}
        onConfirm={makeSelectedSensitive}
        triggerRef={sensitiveButtonRef}
        title="Mark as sensitive"
        description={`This will permanently hide the values of ${plural(selectedCount, "variable")}. They cannot be revealed afterwards. This action cannot be undone.`}
        confirmButtonText="Mark as sensitive"
        cancelButtonText="Cancel"
        variant="danger"
      />

      <ConfirmPopover
        isOpen={isDeleteConfirmOpen}
        onOpenChange={setIsDeleteConfirmOpen}
        onConfirm={deleteSelected}
        triggerRef={deleteButtonRef}
        title="Confirm deletion"
        description={`This will permanently delete ${plural(selectedCount, "environment variable")}. This action cannot be undone.`}
        confirmButtonText={selectedCount === 1 ? "Delete variable" : "Delete variables"}
        cancelButtonText="Cancel"
        variant="danger"
      />
    </>
  );
}
