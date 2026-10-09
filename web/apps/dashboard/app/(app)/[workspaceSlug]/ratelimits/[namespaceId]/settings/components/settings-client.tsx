"use client";

import { collection } from "@/lib/collections";
import { eq, useLiveQuery } from "@tanstack/react-db";
import {
  CopyInput,
  EmptyState,
  EmptyStateActions,
  EmptyStateDescription,
  EmptyStateHeader,
  EmptyStateTitle,
  Input,
  SettingsDangerZone,
  SettingsForm,
  SettingsGroup,
  SettingsGroupContent,
  SettingsRow,
  SettingsZoneRow,
  formSaveState,
  toast,
} from "@unkey/ui";
import { useEffect, useState } from "react";
import { CreateNamespaceButton } from "../../../_components/create-namespace-button";
import { DeleteNamespaceDialog } from "../../_components/namespace-delete-dialog";
import { SettingsClientSkeleton } from "./skeleton";

type Props = {
  namespaceId: string;
};

export const SettingsClient = ({ namespaceId }: Props) => {
  const [isNamespaceNameDeleteModalOpen, setIsNamespaceNameDeleteModalOpen] = useState(false);

  const { data, isLoading } = useLiveQuery((q) =>
    q
      .from({ namespace: collection.ratelimitNamespaces })
      .where(({ namespace }) => eq(namespace.id, namespaceId)),
  );

  const namespace = data.at(0);

  const [namespaceName, setNamespaceName] = useState<string | null>(null);
  const [isSaving, setIsSaving] = useState(false);
  useEffect(() => {
    if (namespaceName === null && namespace) {
      setNamespaceName(namespace.name);
    }
  }, [namespace, namespaceName]);

  if (isLoading) {
    return <SettingsClientSkeleton />;
  }

  if (!namespace) {
    return (
      <EmptyState>
        <EmptyStateHeader>
          <EmptyStateTitle>404</EmptyStateTitle>
          <EmptyStateDescription>This namespace does not exist</EmptyStateDescription>
        </EmptyStateHeader>
        <EmptyStateActions>
          <CreateNamespaceButton />
        </EmptyStateActions>
      </EmptyState>
    );
  }

  const isDirty = namespaceName !== null && namespaceName.trim() !== namespace.name.trim();

  const handleUpdateName = () => {
    if (!isDirty || !namespaceName) {
      return toast.error("Please provide a different name before saving.");
    }
    let error = "";
    collection.ratelimitNamespaces.forEach((ns) => {
      if (ns.id !== namespaceId && namespaceName === ns.name) {
        error = "Another namespace already has this name";
        return;
      }
    });
    if (error) {
      return toast.error(error);
    }

    setIsSaving(true);
    const tx = collection.ratelimitNamespaces.update(namespace.id, (draft) => {
      draft.name = namespaceName;
    });
    const stopSaving = () => setIsSaving(false);
    const failureAlreadyToastedByCollection = stopSaving;
    tx.isPersisted.promise.then(stopSaving, failureAlreadyToastedByCollection);
  };

  return (
    <>
      <SettingsGroup>
        <SettingsGroupContent>
          <SettingsForm
            dirty={isDirty}
            onSubmit={(e) => {
              e.preventDefault();
              handleUpdateName();
            }}
            saveState={formSaveState({
              isSubmitting: isSaving,
              isValid: Boolean(namespaceName),
              isDirty,
            })}
          >
            <SettingsRow
              title="Namespace name"
              description="Used in API calls. Changing this may cause rate limit requests to be rejected."
            >
              <Input
                aria-label="Namespace name"
                placeholder="Namespace name"
                value={namespaceName ?? ""}
                onChange={(e) => setNamespaceName(e.target.value)}
              />
            </SettingsRow>
          </SettingsForm>
          <SettingsRow
            title="Namespace ID"
            description="An identifier for the namespace, used in some API calls."
          >
            <CopyInput value={namespace.id} aria-label="Namespace ID" />
          </SettingsRow>
        </SettingsGroupContent>
      </SettingsGroup>

      <SettingsDangerZone>
        <SettingsZoneRow
          title="Delete ratelimit"
          description="Deletes this namespace along with all associated identifiers and data. This action cannot be undone."
          action={{
            label: "Delete Namespace",
            onClick: () => setIsNamespaceNameDeleteModalOpen(true),
          }}
        />
      </SettingsDangerZone>
      <DeleteNamespaceDialog
        namespace={namespace}
        onOpenChange={setIsNamespaceNameDeleteModalOpen}
        isModalOpen={isNamespaceNameDeleteModalOpen}
      />
    </>
  );
};
