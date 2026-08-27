"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { trpc } from "@/lib/trpc/client";
import { ArrowDottedRotateAnticlockwise, Trash, TriangleWarning2 } from "@unkey/icons";
import { match } from "@unkey/match";
import {
  AlertBanner,
  AlertBannerDescription,
  AlertBannerTitle,
  Button,
  FormInput,
  Skeleton,
  SlidePanel,
  SlidePanelCloseButton,
  SlidePanelContent,
  SlidePanelFooter,
  SlidePanelHeader,
  SlidePanelTitle,
  toast,
} from "@unkey/ui";
import { useRef, useState } from "react";
import { FormProvider } from "react-hook-form";
import { DeleteRootKey } from "../table/delete-root-key";
import { RotateRootKey } from "../table/rotate-root-key";
import { GrantList } from "./grant-list";
import {
  type EditableRootKeyDraft,
  type LegacyRootKeyDraft,
  useRootKeyDraft,
} from "./hooks/use-root-key-draft";
import { useRootKeyPolicyForm } from "./hooks/use-root-key-policy-form";
import { KeyFields } from "./key-fields";
import { buildUrns } from "./lib/urn";
import { PolicyList } from "./policy-list";

type EditKeyAsideProps = {
  keyId: string;
  isOpen: boolean;
  onClose: () => void;
  onExitComplete: () => void;
};

export function EditKeyAside({ keyId, isOpen, onClose, onExitComplete }: EditKeyAsideProps) {
  const draft = useRootKeyDraft(keyId, isOpen, onClose);
  const [requestedAction, setRequestedAction] = useState<KeyAction | null>(null);
  const [openAction, setOpenAction] = useState<KeyAction | null>(null);
  const acted = useRef(false);

  const closeAction = () => {
    setOpenAction(null);
    if (acted.current) {
      onClose();
      onExitComplete();
      return;
    }
    setRequestedAction(null);
  };

  return (
    <>
      <SlidePanel
        isOpen={isOpen && requestedAction === null}
        onClose={onClose}
        onExitComplete={() => {
          if (requestedAction) {
            setOpenAction(requestedAction);
            return;
          }
          onExitComplete();
        }}
        widthClassName="w-192"
      >
        <SlidePanelHeader className="items-center">
          <SlidePanelTitle>Edit key</SlidePanelTitle>
          <SlidePanelCloseButton />
        </SlidePanelHeader>

        <SlidePanelContent>
          {draft === null ? (
            <DraftSkeleton />
          ) : (
            match(draft)
              .with({ kind: "editable" }, (editable) => (
                <EditablePolicyForm
                  draft={editable}
                  onAction={setRequestedAction}
                  onSaved={onClose}
                />
              ))
              .with({ kind: "legacy" }, (legacy) => (
                <LegacyKeyView draft={legacy} onAction={setRequestedAction} />
              ))
              .exhaustive()
          )}
        </SlidePanelContent>
      </SlidePanel>

      {draft !== null && openAction === "rotate" ? (
        <RotateRootKey
          rootKeyDetails={{ id: draft.keyId, name: draft.name === "" ? null : draft.name }}
          isOpen
          onRotated={() => {
            acted.current = true;
          }}
          onClose={closeAction}
        />
      ) : null}

      {draft !== null && openAction === "delete" ? (
        <DeleteRootKey
          rootKeyDetails={{ id: draft.keyId, name: draft.name === "" ? null : draft.name }}
          isOpen
          onDeleted={() => {
            acted.current = true;
          }}
          onClose={closeAction}
        />
      ) : null}
    </>
  );
}

type KeyAction = "rotate" | "delete";

type ActionProps = { onAction: (action: KeyAction) => void };

function EditablePolicyForm({
  draft,
  onAction,
  onSaved,
}: { draft: EditableRootKeyDraft; onSaved: () => void } & ActionProps) {
  const workspace = useWorkspaceNavigation();
  const trpcUtils = trpc.useUtils();
  const updatePermissions = trpc.rootKey.update.permissions.useMutation();
  const updateName = trpc.rootKey.update.name.useMutation();

  const { form, bodyRef, submit } = useRootKeyPolicyForm(
    { name: draft.name, policies: draft.policies },
    async (values) => {
      try {
        await updatePermissions.mutateAsync({
          keyId: draft.keyId,
          permissions: buildUrns(workspace.id, values.policies),
        });
        if (values.name !== draft.name) {
          await updateName.mutateAsync({ keyId: draft.keyId, name: values.name });
        }
      } catch (error) {
        toast.error(error instanceof Error ? error.message : "Failed to update the Root Key.");
        return;
      }

      toast.success("Root Key updated");
      trpcUtils.settings.rootKeys.query.invalidate();
      trpcUtils.settings.rootKeys.get.invalidate({ keyId: draft.keyId });
      onSaved();
    },
  );

  const isSaving = updatePermissions.isLoading || updateName.isLoading;

  return (
    <FormProvider {...form}>
      <form onSubmit={submit} className="flex h-full flex-col">
        <div ref={bodyRef} className="flex-1 overflow-y-auto px-6 py-3">
          <KeyFields>
            <PolicyList />
          </KeyFields>
        </div>
        <SlidePanelFooter className="flex items-center justify-between">
          <ActionButtons onAction={onAction} />
          <Button type="submit" variant="primary" size="md" loading={isSaving} disabled={isSaving}>
            Save changes
          </Button>
        </SlidePanelFooter>
      </form>
    </FormProvider>
  );
}

function LegacyKeyView({ draft, onAction }: { draft: LegacyRootKeyDraft } & ActionProps) {
  return (
    <div className="flex h-full flex-col">
      <div className="flex flex-1 flex-col gap-6 overflow-y-auto px-6 py-3">
        <MigrationNotice />
        <FormInput label="Name" value={draft.name} readOnly />
        <div className="flex flex-col gap-2">
          <span className="flex h-5 items-center text-[13px] text-gray-11">Permissions</span>
          <GrantList grants={draft.grants} />
        </div>
      </div>
      <SlidePanelFooter className="flex items-center justify-between">
        <ActionButtons onAction={onAction} />
        <Button variant="primary" size="md" disabled>
          Save changes
        </Button>
      </SlidePanelFooter>
    </div>
  );
}

function ActionButtons({ onAction }: ActionProps) {
  return (
    <div className="flex items-center gap-2">
      <Button
        type="button"
        variant="outline"
        color="danger"
        size="md"
        onClick={() => onAction("delete")}
      >
        <Trash iconSize="md-medium" />
        Delete key
      </Button>
      <Button type="button" variant="outline" size="md" onClick={() => onAction("rotate")}>
        <ArrowDottedRotateAnticlockwise iconSize="md-medium" />
        Rotate key
      </Button>
    </div>
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
