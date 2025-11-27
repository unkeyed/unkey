import { ExpirationSetup } from "@/app/(app)/[workspaceSlug]/apis/[apiId]/_components/create-key/components/expiration-setup";
import {
  type ExpirationFormValues,
  expirationSchema,
} from "@/app/(app)/[workspaceSlug]/apis/[apiId]/_components/create-key/create-key.schema";
import type { ActionComponentProps } from "@/components/logs/table-action.popover";
import { usePersistedForm } from "@/hooks/use-persisted-form";
import type { KeyDetails } from "@/lib/trpc/routers/api/keys/query-api-keys/schema";
import { zodResolver } from "@hookform/resolvers/zod";
import { Button, DialogContainer } from "@unkey/ui";
import { useEffect } from "react";
import { FormProvider } from "react-hook-form";
import { useEditExpiration } from "../hooks/use-edit-expiration";
import { KeyInfo } from "../key-info";
import { getKeyExpirationDefaults } from "./utils";

const EDIT_EXPIRATION_FORM_STORAGE_KEY = "unkey_edit_expiration_form_state";

type EditExpirationProps = {
  keyDetails: KeyDetails;
} & ActionComponentProps;

export const EditExpiration = ({ keyDetails, isOpen, onClose }: EditExpirationProps) => {
  const methods = usePersistedForm<ExpirationFormValues>(
    `${EDIT_EXPIRATION_FORM_STORAGE_KEY}_${keyDetails.id}`,
    {
      resolver: zodResolver(expirationSchema),
      mode: "onChange",
      shouldFocusError: true,
      shouldUnregister: true,
      defaultValues: getKeyExpirationDefaults(keyDetails),
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

  const updateExpiration = useEditExpiration(() => {
    reset(getKeyExpirationDefaults(keyDetails));
    clearPersistedData();
    onClose();
  });

  const onSubmit = async (data: ExpirationFormValues) => {
    try {
      await updateExpiration.mutateAsync({
        keyId: keyDetails.id,
        expiration: {
          enabled: data.expiration.enabled,
          data: data.expiration.enabled ? data.expiration.data : undefined,
        },
      });
    } catch {
      // `useEditExpiration` already shows a toast, but we still need to
      // prevent unhandled rejection noise in the console.
    }
  };

  return (
    <FormProvider {...methods}>
      <form id="edit-expiration-form" onSubmit={handleSubmit(onSubmit)}>
        <DialogContainer
          isOpen={isOpen}
          subTitle="Automatically revoke this key after a certain date"
          onOpenChange={() => {
            saveCurrentValues();
            onClose();
          }}
          title="Edit expiration"
          footer={
            <div className="w-full flex flex-col gap-2 items-center justify-center">
              <Button
                type="submit"
                form="edit-expiration-form"
                variant="primary"
                size="xlg"
                className="w-full rounded-lg"
                disabled={!isValid || isSubmitting}
                loading={updateExpiration.isPending}
              >
                Update expiration
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
            <ExpirationSetup />
          </div>
        </DialogContainer>
      </form>
    </FormProvider>
  );
};
