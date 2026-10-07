"use client";

import type { ActionComponentProps } from "@/components/logs/table-action.popover";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { deleteRootKey, rootKeysV2QueryKeys } from "@/lib/root-keys-api";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Button, Dialog, DialogContent, DialogTitle, FormCheckbox, toast } from "@unkey/ui";
import { useState } from "react";

type DeleteRootKeyProps = {
  rootKeyDetails: { id: string; name: string | null };
  onDeleted?: () => void;
} & ActionComponentProps;

export function DeleteRootKey({ rootKeyDetails, isOpen, onClose, onDeleted }: DeleteRootKeyProps) {
  const workspace = useWorkspaceNavigation();
  const queryClient = useQueryClient();
  const mutation = useMutation({
    mutationFn: async ({ keyIds }: { keyIds: string[] }) => {
      await Promise.all(keyIds.map((keyId) => deleteRootKey({ keyId })));
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: rootKeysV2QueryKeys.workspace(workspace.id),
      });
      toast.success("Root Key deleted", {
        description:
          "The Root Key has been permanently deleted and can no longer create resources.",
      });
      onDeleted?.();
      onClose();
    },
    onError: (error) => {
      toast.error("Failed to revoke Root Key", {
        description: error instanceof Error ? error.message : "Please try again later.",
      });
    },
  });

  return (
    <DeleteRootKeyDialog
      rootKeyDetails={rootKeyDetails}
      isOpen={isOpen}
      onClose={onClose}
      deleteRootKey={mutation}
    />
  );
}

type DeleteRootKeyDialogProps = Omit<DeleteRootKeyProps, "onDeleted"> & {
  deleteRootKey: {
    mutateAsync: (input: { keyIds: string[] }) => Promise<unknown>;
  };
};

function DeleteRootKeyDialog({
  rootKeyDetails,
  isOpen,
  onClose,
  deleteRootKey,
}: DeleteRootKeyDialogProps) {
  const [isConfirmed, setIsConfirmed] = useState(false);
  const [isLoading, setIsLoading] = useState(false);

  const remove = async () => {
    try {
      setIsLoading(true);
      await deleteRootKey.mutateAsync({ keyIds: [rootKeyDetails.id] });
    } catch {
      // The mutation hook surfaces its own toast.
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <Dialog
      open={isOpen}
      onOpenChange={(open) => {
        if (!open && !isLoading) {
          onClose();
        }
      }}
    >
      <DialogContent
        data-docs-target="root-key-delete"
        className="w-full max-w-[560px] gap-4 rounded-2xl! border-grayA-4 p-6"
      >
        <div className="flex flex-col gap-1">
          <DialogTitle>
            {rootKeyDetails.name ? `Delete “${rootKeyDetails.name}”?` : "Delete this root key?"}
          </DialogTitle>
          <p className="text-sm leading-5 text-gray-11">
            This cannot be undone. The key stops working within 60 seconds and anything still using
            it will fail.
          </p>
        </div>
        <FormCheckbox
          size="lg"
          checked={isConfirmed}
          onCheckedChange={(next) => setIsConfirmed(next === true)}
          label="I understand this will permanently delete the Root Key."
        />
        <div className="flex items-center justify-end gap-2 pt-1">
          <Button type="button" variant="outline" size="md" onClick={onClose} disabled={isLoading}>
            Cancel
          </Button>
          <Button
            type="button"
            variant="primary"
            color="danger"
            size="md"
            onClick={remove}
            disabled={!isConfirmed || isLoading}
            loading={isLoading}
          >
            Delete key
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  );
}
