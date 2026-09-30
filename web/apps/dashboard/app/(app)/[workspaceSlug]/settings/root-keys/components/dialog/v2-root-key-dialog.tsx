"use client";

import { ROOT_KEYS_V2_QUERY_KEY, updateRootKey } from "@/lib/root-keys-api";
import type { RootKey } from "@/lib/trpc/routers/settings/root-keys/query";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Button, DialogContainer, FormInput, toast } from "@unkey/ui";
import { useState } from "react";

type V2RootKeyDialogProps = {
  rootKey: RootKey;
  isOpen: boolean;
  onOpenChange: (open: boolean) => void;
};

export function V2RootKeyDialog({ rootKey, isOpen, onOpenChange }: V2RootKeyDialogProps) {
  const queryClient = useQueryClient();
  const [name, setName] = useState(rootKey.name ?? "");

  const mutation = useMutation({
    mutationFn: () =>
      updateRootKey({
        keyId: rootKey.id,
        name: name.trim() || null,
      }),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ROOT_KEYS_V2_QUERY_KEY });
      toast.success("Root key updated");
      onOpenChange(false);
    },
    onError: (error) => {
      toast.error("Failed to update root key", {
        description: error instanceof Error ? error.message : "Please try again.",
      });
    },
  });

  const normalizedName = name.trim();
  const hasChanges = normalizedName !== (rootKey.name ?? "");

  return (
    <DialogContainer
      data-docs-target="root-key-edit"
      isOpen={isOpen}
      onOpenChange={onOpenChange}
      title="Edit root key"
      subTitle="Update the name for this root key"
      footer={
        <Button
          variant="primary"
          size="xlg"
          className="w-full rounded-lg"
          disabled={!hasChanges || mutation.isLoading}
          loading={mutation.isLoading}
          onClick={() => mutation.mutate()}
        >
          Update root key
        </Button>
      }
    >
      <FormInput
        name="name"
        label="Name"
        description="A name that helps your team identify this root key."
        placeholder="e.g. CI deploy key"
        value={name}
        onChange={(event) => setName(event.target.value)}
      />
    </DialogContainer>
  );
}
