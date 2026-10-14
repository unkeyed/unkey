"use client";

import { useInvalidateNamespaces, useNamespace } from "@/lib/queries/ratelimit-namespaces";
import { getErrorMessage, getUnkeyClient } from "@/lib/unkey-client";
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
  SettingsRowContent,
  SettingsRowDescription,
  SettingsRowHeader,
  SettingsRowTitle,
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
  const invalidateNamespaces = useInvalidateNamespaces();

  const { data: namespace, isLoading } = useNamespace(namespaceId);

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

  const handleUpdateName = async () => {
    if (!isDirty || !namespaceName) {
      return toast.error("Please provide a different name before saving.");
    }
    setIsSaving(true);
    try {
      const mutation = getUnkeyClient().ratelimit.updateNamespace({
        namespace: namespaceId,
        name: namespaceName,
      });
      toast.promise(mutation, {
        loading: "Updating namespace...",
        success: "Namespace updated",
        error: (err) => ({
          message: "Failed to update namespace",
          description: getErrorMessage(err),
        }),
      });
      await mutation;
      await invalidateNamespaces();
    } catch (error) {
      console.error("Failed to update namespace", error);
    } finally {
      setIsSaving(false);
    }
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
            <SettingsRow>
              <SettingsRowHeader>
                <SettingsRowTitle>Namespace name</SettingsRowTitle>
                <SettingsRowDescription>
                  Used in API calls. Changing this may cause rate limit requests to be rejected.
                </SettingsRowDescription>
              </SettingsRowHeader>
              <SettingsRowContent>
                <Input
                  aria-label="Namespace name"
                  placeholder="Namespace name"
                  value={namespaceName ?? ""}
                  className="max-w-(--setting-w)"
                  onChange={(e) => setNamespaceName(e.target.value)}
                />
              </SettingsRowContent>
            </SettingsRow>
          </SettingsForm>
          <SettingsRow>
            <SettingsRowHeader>
              <SettingsRowTitle>Namespace ID</SettingsRowTitle>
              <SettingsRowDescription>
                An identifier for the namespace, used in some API calls.
              </SettingsRowDescription>
            </SettingsRowHeader>
            <SettingsRowContent>
              <CopyInput
                value={namespace.id}
                aria-label="Namespace ID"
                className="max-w-(--setting-w)"
              />
            </SettingsRowContent>
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
