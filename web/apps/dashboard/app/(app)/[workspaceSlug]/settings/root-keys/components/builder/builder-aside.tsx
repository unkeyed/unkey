"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import { Button, SlidePanel, toast } from "@unkey/ui";
import { useRef, useState } from "react";
import { FormProvider, useForm } from "react-hook-form";
import { KeyFields } from "./key-fields";
import { buildUrns } from "./lib/urn";
import { PolicyList } from "./policy-list";
import { type RootKeyFormValues, rootKeyDefaultValues, rootKeySchema } from "./schema";
import { SuccessDialog } from "./success-dialog";

type BuilderAsideProps = {
  isOpen: boolean;
  onClose: () => void;
};

export function BuilderAside({ isOpen, onClose }: BuilderAsideProps) {
  const workspace = useWorkspaceNavigation();
  const trpcUtils = trpc.useUtils();
  const [secret, setSecret] = useState<string | null>(null);
  const [secretRevealed, setSecretRevealed] = useState(false);
  const form = useForm<RootKeyFormValues>({
    resolver: zodResolver(rootKeySchema),
    defaultValues: rootKeyDefaultValues,
  });
  const bodyRef = useRef<HTMLDivElement>(null);

  const createKey = trpc.rootKey.create.useMutation({
    onSuccess(data) {
      trpcUtils.settings.rootKeys.query.invalidate();
      setSecret(data.key);
    },
    onError(err) {
      if (err.data?.code === "BAD_REQUEST") {
        toast.error("You need to add at least one permission.");
        return;
      }
      toast.error(err.message);
    },
  });

  const scrollToTop = () => {
    bodyRef.current?.scrollTo({
      top: 0,
      behavior: window.matchMedia("(prefers-reduced-motion: reduce)").matches ? "auto" : "smooth",
    });
  };

  const submit = form.handleSubmit((values) => {
    createKey.mutate({
      name: values.name,
      permissions: buildUrns(workspace.id, values.policies),
    });
  }, scrollToTop);

  const finish = () => {
    setSecret(null);
    setSecretRevealed(false);
    form.reset(rootKeyDefaultValues);
    onClose();
  };

  return (
    <>
      <SlidePanel.Root
        isOpen={isOpen && secret === null}
        onClose={onClose}
        onExitComplete={() => {
          if (secret !== null) {
            setSecretRevealed(true);
          }
        }}
        widthClassName="w-192"
      >
        <SlidePanel.Header>
          <div className="flex flex-col gap-0.5">
            <SlidePanel.Title>New root key</SlidePanel.Title>
            <SlidePanel.Description>Name the key and grant it permissions.</SlidePanel.Description>
          </div>
          <SlidePanel.CloseButton className="mt-0.5" />
        </SlidePanel.Header>

        <SlidePanel.Content>
          <FormProvider {...form}>
            <form onSubmit={submit} className="flex h-full flex-col">
              <div ref={bodyRef} className="flex-1 overflow-y-auto px-6 py-3">
                <KeyFields>
                  <PolicyList />
                </KeyFields>
              </div>

              <SlidePanel.Footer className="flex items-center justify-end">
                <Button
                  type="submit"
                  variant="primary"
                  size="md"
                  loading={createKey.isLoading}
                  disabled={createKey.isLoading}
                >
                  Create key
                </Button>
              </SlidePanel.Footer>
            </form>
          </FormProvider>
        </SlidePanel.Content>
      </SlidePanel.Root>

      {secret !== null && secretRevealed ? <SuccessDialog secret={secret} onDone={finish} /> : null}
    </>
  );
}
