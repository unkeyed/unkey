"use client";

import { IdentityInfo } from "@/app/(app)/[workspaceSlug]/identities/_components/dialogs/identity-info";
import { MetadataSetup } from "@/components/dashboard/metadata/metadata-setup";
import type { ActionComponentProps } from "@/components/logs/table-action.popover";
import { usePersistedForm } from "@/hooks/use-persisted-form";
import { type MetadataFormValues, metadataSchema } from "@/lib/schemas/metadata";
import type { DiscriminatedUnionResolver } from "@/lib/schemas/resolver-types";
import { trpc } from "@/lib/trpc/client";
import type { IdentityResponseSchema } from "@/lib/trpc/routers/identity/query";
import { zodResolver } from "@hookform/resolvers/zod";
import { Button, DialogContainer, toast } from "@unkey/ui";
import { type FC, useEffect, useId } from "react";
import { FormProvider } from "react-hook-form";
import type { z } from "zod";

type Identity = z.infer<typeof IdentityResponseSchema>;

type EditMetadataDialogProps = { identity: Identity } & ActionComponentProps;

const EDIT_METADATA_FORM_STORAGE_KEY = "unkey_edit_identity_metadata_form_state";

const getIdentityMetadataDefaults = (identity: Identity) => ({
  metadata: identity.meta
    ? ({
        enabled: true as const,
        data: JSON.stringify(identity.meta, null, 2),
      } as const)
    : ({ enabled: false as const } as const),
});

export const EditMetadataDialog: FC<EditMetadataDialogProps> = ({ identity, isOpen, onClose }) => {
  const formId = useId();
  const utils = trpc.useUtils();

  const methods = usePersistedForm<MetadataFormValues>(
    `${EDIT_METADATA_FORM_STORAGE_KEY}_${identity.id}`,
    {
      resolver: zodResolver(metadataSchema) as DiscriminatedUnionResolver<typeof metadataSchema>,
      mode: "onChange",
      shouldFocusError: true,
      shouldUnregister: true,
      defaultValues: getIdentityMetadataDefaults(identity),
    },
    "memory",
  );

  const {
    handleSubmit,
    formState: { isSubmitting, isValid },
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

  const updateMetadata = trpc.identity.update.metadata.useMutation({
    onSuccess: () => {
      toast.success("Metadata updated successfully");
      utils.identity.query.invalidate();
      utils.identity.getById.invalidate();
      reset(getIdentityMetadataDefaults(identity));
      clearPersistedData();
      onClose();
    },
    onError: (error) => {
      toast.error(error.message || "Failed to update metadata");
    },
  });

  const onSubmit = async (data: MetadataFormValues) => {
    try {
      await updateMetadata.mutateAsync({
        identityId: identity.id,
        metadata: data.metadata,
      });
    } catch {
      // updateMetadata.onError already shows a toast.
    }
  };

  return (
    <FormProvider {...methods}>
      <form id={formId} onSubmit={handleSubmit(onSubmit)}>
        <DialogContainer
          isOpen={isOpen}
          onOpenChange={(o) => {
            if (!o) {
              saveCurrentValues();
              onClose();
            }
          }}
          title="Edit metadata"
          subTitle="Attach custom data to this identity"
          footer={
            <div className="w-full flex flex-col gap-2 items-center justify-center">
              <Button
                type="submit"
                form={formId}
                variant="primary"
                size="xlg"
                className="w-full rounded-lg"
                disabled={!isValid || isSubmitting}
                loading={updateMetadata.isLoading}
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
