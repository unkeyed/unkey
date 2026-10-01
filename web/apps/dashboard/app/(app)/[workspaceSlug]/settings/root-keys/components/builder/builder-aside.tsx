"use client";

import { SecretKeyDialog } from "@/components/secret-key-dialog";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { createRootKey, rootKeysV2QueryKeys } from "@/lib/root-keys-api";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  Button,
  SlidePanel,
  SlidePanelCloseButton,
  SlidePanelContent,
  SlidePanelFooter,
  SlidePanelHeader,
  SlidePanelTitle,
  toast,
} from "@unkey/ui";
import { useState } from "react";
import { FormProvider } from "react-hook-form";
import { useRootKeyPolicyForm } from "./hooks/use-root-key-policy-form";
import { KeyFields } from "./key-fields";
import { buildUrns } from "./lib/urn";
import { PolicyList } from "./policy-list";
import { rootKeyDefaultValues } from "./schema";

type BuilderAsideProps = {
  isOpen: boolean;
  onClose: () => void;
};

export function BuilderAside({ isOpen, onClose }: BuilderAsideProps) {
  const workspace = useWorkspaceNavigation();
  const queryClient = useQueryClient();
  const [secret, setSecret] = useState<string | null>(null);
  const [secretRevealed, setSecretRevealed] = useState(false);

  const createKey = useMutation({
    mutationFn: createRootKey,
    onSuccess(data) {
      void queryClient.invalidateQueries({ queryKey: rootKeysV2QueryKeys.workspace(workspace.id) });
      setSecret(data.key);
    },
    onError(error) {
      toast.error(error instanceof Error ? error.message : "Failed to create the Root Key.");
    },
  });

  const { form, bodyRef, submit } = useRootKeyPolicyForm(rootKeyDefaultValues, (values) => {
    createKey.mutate({
      name: values.name.trim() || undefined,
      permissions: buildUrns(workspace.id, values.policies),
    });
  });

  const finish = () => {
    setSecret(null);
    setSecretRevealed(false);
    form.reset(rootKeyDefaultValues);
    onClose();
  };

  const close = () => {
    if (!createKey.isLoading) {
      onClose();
    }
  };

  return (
    <>
      <SlidePanel
        isOpen={isOpen && secret === null}
        onClose={close}
        onExitComplete={() => {
          if (secret !== null) {
            setSecretRevealed(true);
          }
        }}
        widthClassName="w-192"
      >
        <SlidePanelHeader className="items-center">
          <SlidePanelTitle>New Root Key</SlidePanelTitle>
          <SlidePanelCloseButton disabled={createKey.isLoading} />
        </SlidePanelHeader>

        <SlidePanelContent>
          <FormProvider {...form}>
            <form onSubmit={submit} className="flex h-full flex-col">
              <div ref={bodyRef} className="flex-1 overflow-y-auto px-6 py-3">
                <KeyFields>
                  <PolicyList />
                </KeyFields>
              </div>

              <SlidePanelFooter className="flex items-center justify-end">
                <Button
                  type="submit"
                  variant="primary"
                  size="md"
                  loading={createKey.isLoading}
                  disabled={createKey.isLoading}
                >
                  Create key
                </Button>
              </SlidePanelFooter>
            </form>
          </FormProvider>
        </SlidePanelContent>
      </SlidePanel>

      {secret !== null && secretRevealed ? (
        <SecretKeyDialog secret={secret} onDone={finish} />
      ) : null}
    </>
  );
}
