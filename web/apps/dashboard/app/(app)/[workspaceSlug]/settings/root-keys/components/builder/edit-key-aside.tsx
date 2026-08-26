"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { TriangleWarning2 } from "@unkey/icons";
import { match } from "@unkey/match";
import {
  AlertBanner,
  AlertBannerDescription,
  AlertBannerTitle,
  Button,
  Skeleton,
  SlidePanel,
  toast,
} from "@unkey/ui";
import { useState } from "react";
import { FormProvider, useForm } from "react-hook-form";
import { RotateRootKey } from "../table/rotate-root-key";
import { GrantList } from "./grant-list";
import {
  type EditableRootKeyDraft,
  type LegacyRootKeyDraft,
  useRootKeyDraft,
} from "./hooks/use-root-key-draft";
import { useRootKeyPolicyForm } from "./hooks/use-root-key-policy-form";
import { KeyFields, NameField } from "./key-fields";
import { PolicyList } from "./policy-list";
import { type LegacyRootKeyFormValues, legacyRootKeySchema } from "./schema";

// TODO: swap for the update-root-key mutation once the backend lane accepts URN grants.
function reportUnwired() {
  toast.info("Saving isn't wired up yet");
}

type EditKeyAsideProps = {
  keyId: string;
  isOpen: boolean;
  onClose: () => void;
  onExitComplete: () => void;
};

export function EditKeyAside({ keyId, isOpen, onClose, onExitComplete }: EditKeyAsideProps) {
  const draft = useRootKeyDraft(keyId, isOpen, onClose);
  // The rotate dialog sits at z-50 and the panel at z-51, so the two cannot
  // share the screen. Slide the panel out first, open the dialog once it has
  // gone, and slide it back when the dialog closes — the same handoff the
  // create aside uses for its success dialog.
  const [rotateRequested, setRotateRequested] = useState(false);
  const [rotateOpen, setRotateOpen] = useState(false);

  const requestRotate = () => setRotateRequested(true);
  const finishRotate = () => {
    setRotateOpen(false);
    setRotateRequested(false);
  };

  return (
    <>
      <SlidePanel.Root
        isOpen={isOpen && !rotateRequested}
        onClose={onClose}
        onExitComplete={() => {
          if (rotateRequested) {
            setRotateOpen(true);
            return;
          }
          onExitComplete();
        }}
        widthClassName="w-192"
      >
        <SlidePanel.Header className="items-center">
          <SlidePanel.Title>Edit key</SlidePanel.Title>
          <SlidePanel.CloseButton />
        </SlidePanel.Header>

        <SlidePanel.Content>
          {draft === null ? (
            <DraftSkeleton />
          ) : (
            match(draft)
              .with({ kind: "editable" }, (editable) => (
                <EditablePolicyForm draft={editable} onRotate={requestRotate} />
              ))
              .with({ kind: "legacy" }, (legacy) => (
                <LegacyKeyView draft={legacy} onRotate={requestRotate} />
              ))
              .exhaustive()
          )}
        </SlidePanel.Content>
      </SlidePanel.Root>

      {draft !== null && rotateOpen ? (
        <RotateRootKey
          rootKeyDetails={{
            id: draft.keyId,
            start: draft.start,
            name: draft.name === "" ? null : draft.name,
          }}
          isOpen
          onClose={finishRotate}
        />
      ) : null}
    </>
  );
}

type RotateProps = { onRotate: () => void };

function EditablePolicyForm({ draft, onRotate }: { draft: EditableRootKeyDraft } & RotateProps) {
  const { form, bodyRef, submit } = useRootKeyPolicyForm(
    { name: draft.name, policies: draft.policies },
    reportUnwired,
  );

  return (
    <FormProvider {...form}>
      <form onSubmit={submit} className="flex h-full flex-col">
        <div ref={bodyRef} className="flex-1 overflow-y-auto px-6 py-3">
          <KeyFields>
            <PolicyList />
          </KeyFields>
        </div>
        <SaveFooter onRotate={onRotate} />
      </form>
    </FormProvider>
  );
}

function LegacyKeyView({ draft, onRotate }: { draft: LegacyRootKeyDraft } & RotateProps) {
  const form = useForm<LegacyRootKeyFormValues>({
    resolver: zodResolver(legacyRootKeySchema),
    defaultValues: { name: draft.name },
  });

  return (
    <FormProvider {...form}>
      <form onSubmit={form.handleSubmit(reportUnwired)} className="flex h-full flex-col">
        <div className="flex flex-1 flex-col gap-6 overflow-y-auto px-6 py-3">
          <MigrationNotice />
          <NameField />
          <div className="flex flex-col gap-2">
            <span className="flex h-5 items-center text-[13px] text-gray-11">Permissions</span>
            <GrantList grants={draft.grants} />
          </div>
        </div>
        <SaveFooter onRotate={onRotate} />
      </form>
    </FormProvider>
  );
}

function SaveFooter({ onRotate }: RotateProps) {
  return (
    <SlidePanel.Footer className="flex items-center justify-between">
      <Button type="button" variant="outline" size="md" onClick={onRotate}>
        Rotate key
      </Button>
      <Button type="submit" variant="primary" size="md">
        Save changes
      </Button>
    </SlidePanel.Footer>
  );
}

function MigrationNotice() {
  return (
    <AlertBanner variant="error">
      <TriangleWarning2 iconSize="md-regular" />
      <AlertBannerTitle>Legacy key</AlertBannerTitle>
      <AlertBannerDescription>
        This key was created before the permissions migration. Once the migration takes place, this
        key will become editable and this notice will disappear. For now, this key is view only.
      </AlertBannerDescription>
    </AlertBanner>
  );
}

function DraftSkeleton() {
  return (
    <div className="flex flex-col gap-6 px-6 py-3">
      <div className="flex flex-col gap-2">
        <Skeleton className="h-4 w-14" />
        <Skeleton className="h-9 w-full" />
      </div>
      <div className="flex flex-col gap-3">
        <Skeleton className="h-4 w-24" />
        <Skeleton className="h-[70px] w-full" />
        <Skeleton className="h-[70px] w-full" />
      </div>
    </div>
  );
}
