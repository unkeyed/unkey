"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import type { Identity } from "@unkey/api/models/components";
import { Alert, AlertDescription, AlertTitle, Button, DialogContainer } from "@unkey/ui";
import { type FC, useEffect, useId } from "react";
import { FormProvider } from "react-hook-form";
import { IdentityInfo } from "@/app/(app)/[workspaceSlug]/identities/_components/dialogs/identity-info";
import { MetadataSetup } from "@/components/dashboard/metadata/metadata-setup";
import type { ActionComponentProps } from "@/components/logs/table-action.popover";
import { usePersistedForm } from "@/hooks/use-persisted-form";
import { useUpdateIdentityMutation } from "@/lib/identities-query";
import {
  identityMetadataSchema,
  type MetadataFormValues,
  parseIdentityMetadata,
} from "@/lib/schemas/metadata";
import type { DiscriminatedUnionResolver } from "@/lib/schemas/resolver-types";
import { getErrorMessage } from "@/lib/unkey-client";

type EditMetadataDialogProps = { identity: Identity } & ActionComponentProps;

const EDIT_METADATA_FORM_STORAGE_KEY = "unkey_edit_identity_metadata_form_state";

const getIdentityMetadataDefaults = (identity: Identity) => ({
  metadata:
    identity.meta && Object.keys(identity.meta).length > 0
      ? ({
          enabled: true as const,
          data: JSON.stringify(identity.meta, null, 2),
        } as const)
      : ({ enabled: false as const } as const),
});

export const EditMetadataDialog: FC<EditMetadataDialogProps> = ({ identity, isOpen, onClose }) => {
  const formId = useId();
  const updateIdentity = useUpdateIdentityMutation();

  const methods = usePersistedForm<MetadataFormValues>(
    `${EDIT_METADATA_FORM_STORAGE_KEY}_${identity.id}`,
    {
      resolver: zodResolver(identityMetadataSchema) as DiscriminatedUnionResolver<
        typeof identityMetadataSchema
      >,
      mode: "onChange",
      shouldFocusError: true,
      shouldUnregister: true,
      defaultValues: getIdentityMetadataDefaults(identity),
    },
    "memory",
  );

  const {
    handleSubmit,
    formState: { isDirty, isSubmitting, isValid },
    loadSavedValues,
    saveCurrentValues,
    clearPersistedData,
    reset,
  } = methods;

  useEffect(() => {
    if (isOpen) {
      loadSavedValues();
    }
  }, [isOpen, loadSavedValues]);

  const onSubmit = async (data: MetadataFormValues) => {
    try {
      const value = data.metadata.enabled ? parseIdentityMetadata(data.metadata.data) : {};
      const updatedIdentity = await updateIdentity.mutateAsync({
        identity: identity.id,
        meta: value,
      });
      reset(getIdentityMetadataDefaults(updatedIdentity));
      clearPersistedData();
    } catch {
      // The mutation state keeps the error visible in the dialog.
    }
  };

  const showSuccess = updateIdentity.isSuccess && !isDirty;

  return (
    <FormProvider {...methods}>
      <form id={formId} onSubmit={handleSubmit(onSubmit)}>
        <DialogContainer
          isOpen={isOpen}
          onOpenChange={(o) => {
            if (!o && !isSubmitting) {
              if (showSuccess) {
                clearPersistedData();
              } else {
                saveCurrentValues();
              }
              updateIdentity.reset();
              onClose();
            }
          }}
          title="Edit metadata"
          subTitle="Attach custom data to this identity"
          footer={
            <div className="w-full flex flex-col gap-3 items-center justify-center">
              {updateIdentity.isError ? (
                <Alert variant="alert">
                  <AlertTitle>Couldn&apos;t Update Metadata</AlertTitle>
                  <AlertDescription>
                    {getErrorMessage(updateIdentity.error)} Review your metadata and try again.
                  </AlertDescription>
                </Alert>
              ) : null}
              {showSuccess ? (
                <output
                  aria-live="polite"
                  className="w-full rounded-lg border border-success-7 bg-successA-2 p-4 text-success-11"
                >
                  <span className="block font-medium leading-none">Metadata Updated</span>
                  <span className="mt-1 block text-sm">Your changes are now active.</span>
                </output>
              ) : null}
              <Button
                type="submit"
                form={formId}
                variant="primary"
                size="xlg"
                className="w-full rounded-lg"
                disabled={!isValid || isSubmitting || showSuccess}
                loading={isSubmitting}
              >
                Update metadata
              </Button>
              <div className="text-gray-9 text-xs">Changes will be applied immediately</div>
            </div>
          }
        >
          <IdentityInfo identity={identity} />
          <div className="py-1 my-2">
            <div className="h-px bg-grayA-3 w-full" />
          </div>
          <div className="[&>*:first-child]:p-0">
            <MetadataSetup entityType="identity" />
          </div>
        </DialogContainer>
      </form>
    </FormProvider>
  );
};
