"use client";
import { MetadataSetup } from "@/components/dashboard/metadata/metadata-setup";
import type { ActionComponentProps } from "@/components/logs/table-action.popover";
import { usePersistedForm } from "@/hooks/use-persisted-form";
import { type MetadataFormValues, metadataSchema } from "@/lib/schemas/metadata";
import type { DiscriminatedUnionResolver } from "@/lib/schemas/resolver-types";
import type { KeyDetails } from "@/lib/trpc/routers/api/keys/query-api-keys/schema";
import { zodResolver } from "@hookform/resolvers/zod";
import { Button, DialogContainer } from "@unkey/ui";
import { useEffect } from "react";
import { FormProvider } from "react-hook-form";
import { useEditMetadata } from "../hooks/use-edit-metadata";
import { KeyInfo } from "../key-info";
import { getKeyMetadataDefaults } from "./utils";

const EDIT_METADATA_FORM_STORAGE_KEY = "unkey_edit_metadata_form_state";

type EditMetadataProps = {
  keyDetails: KeyDetails;
} & ActionComponentProps;

export const EditMetadata = ({ keyDetails, isOpen, onClose }: EditMetadataProps) => {
  const methods = usePersistedForm<MetadataFormValues>(
    `${EDIT_METADATA_FORM_STORAGE_KEY}_${keyDetails.id}`,
    {
      resolver: zodResolver(metadataSchema) as DiscriminatedUnionResolver<typeof metadataSchema>,
      mode: "onChange",
      shouldFocusError: true,
      shouldUnregister: true,
      defaultValues: getKeyMetadataDefaults(keyDetails),
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

  // Load saved values when the dialog opens
  useEffect(() => {
    if (isOpen) {
      loadSavedValues();
    }
  }, [isOpen, loadSavedValues]);

  const updateMetadata = useEditMetadata(() => {
    reset(getKeyMetadataDefaults(keyDetails));
    clearPersistedData();
    onClose();
  });

  const onSubmit = async (data: MetadataFormValues) => {
    try {
      if (data.metadata.enabled && data.metadata.data) {
        await updateMetadata.mutateAsync({
          keyId: keyDetails.id,
          metadata: {
            enabled: data.metadata.enabled,
            data: data.metadata.data,
          },
        });
      } else {
        await updateMetadata.mutateAsync({
          keyId: keyDetails.id,
          metadata: {
            enabled: false,
          },
        });
      }
    } catch {
      // useEditMetadata already shows a toast, but we still need to
      // prevent unhandled rejection noise in the console.
    }
  };

  return (
    <FormProvider {...methods}>
      <form id="edit-metadata-form" onSubmit={handleSubmit(onSubmit)}>
        <DialogContainer
          isOpen={isOpen}
          subTitle="Attach custom data to this key"
          onOpenChange={() => {
            saveCurrentValues();
            onClose();
          }}
          title="Edit metadata"
          footer={
            <div className="w-full flex flex-col gap-2 items-center justify-center">
              <Button
                type="submit"
                form="edit-metadata-form"
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
          <KeyInfo keyDetails={keyDetails} />
          <div className="py-1 my-2">
            <div className="h-[1px] bg-grayA-3 w-full" />
          </div>
          <div className="[&>*:first-child]:p-0">
            <MetadataSetup entityType="key" />
          </div>
        </DialogContainer>
      </form>
    </FormProvider>
  );
};
