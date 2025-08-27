import { UsageSetup } from "@/app/(app)/[workspaceId]/apis/[apiId]/_components/create-key/components/credits-setup";
import {
  type CreditsFormValues,
  creditsSchema,
} from "@/app/(app)/[workspaceId]/apis/[apiId]/_components/create-key/create-key.schema";
import type { ActionComponentProps } from "@/components/logs/table-action.popover";
import { usePersistedForm } from "@/hooks/use-persisted-form";
import { trpc } from "@/lib/trpc/client";
import type { KeyDetails } from "@/lib/trpc/routers/api/keys/query-api-keys/schema";
import { zodResolver } from "@hookform/resolvers/zod";
import { Button, DialogContainer, toast } from "@unkey/ui";
import { useEffect } from "react";
import { FormProvider } from "react-hook-form";
import { useEditCredits } from "../hooks/use-edit-credits";
import { KeyInfo } from "../key-info";
import { getKeyLimitDefaults } from "./utils";

const EDIT_CREDITS_FORM_STORAGE_KEY = "unkey_edit_credits_form_state";

type EditCreditsProps = { keyDetails: KeyDetails } & ActionComponentProps;

export const EditCredits = ({ keyDetails, isOpen, onClose }: EditCreditsProps) => {
  const trpcUtil = trpc.useUtils();
  const methods = usePersistedForm<CreditsFormValues>(
    `${EDIT_CREDITS_FORM_STORAGE_KEY}_${keyDetails.id}`,
    {
      resolver: zodResolver(creditsSchema),
      mode: "onChange",
      shouldFocusError: true,
      shouldUnregister: true,
      defaultValues: getKeyLimitDefaults(keyDetails),
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

  const key = useEditCredits(() => {
    reset(getKeyLimitDefaults(keyDetails));
    clearPersistedData();
    trpcUtil.key.fetchPermissions.invalidate();
    onClose();
  });

  const onSubmit = async (data: CreditsFormValues) => {
    try {
      if (data.limit) {
        if (data.limit.enabled === true) {
          if (data.limit.data) {
            await key.mutateAsync({
              keyId: keyDetails.id,
              limit: {
                enabled: true,
                data: data.limit.data,
              },
            });
          } else {
            // Shouldn't happen
            toast.error("Failed to Update Key Limits", {
              description: "An unexpected error occurred. Please try again later.",
              action: {
                label: "Contact Support",
                onClick: () => window.open("https://support.unkey.dev", "_blank"),
              },
            });
          }
        } else {
          await key.mutateAsync({
            keyId: keyDetails.id,
            limit: {
              enabled: false,
            },
          });
        }
      }
    } catch {
      // `useEditKeyRemainingUses` already shows a toast, but we still need to
      // prevent unhandled‐rejection noise in the console.
    }
  };

  return (
    <FormProvider {...methods}>
      <form id="edit-remaining-uses-form" onSubmit={handleSubmit(onSubmit)}>
        <DialogContainer
          isOpen={isOpen}
          subTitle="Update the number of credits and refill settings for this key"
          onOpenChange={() => {
            saveCurrentValues();
            onClose();
          }}
          title="Edit Credits"
          footer={
            <div className="w-full flex flex-col gap-2 items-center justify-center">
              <Button
                type="submit"
                form="edit-remaining-uses-form"
                variant="primary"
                size="xlg"
                className="w-full rounded-lg"
                disabled={!isValid || isSubmitting}
                loading={key.isLoading}
              >
                Update credits
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
            <UsageSetup />
          </div>
        </DialogContainer>
      </form>
    </FormProvider>
  );
};
