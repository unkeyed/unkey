import { RatelimitSetup } from "@/components/dashboard/ratelimits/ratelimit-setup";
import type { ActionComponentProps } from "@/components/logs/table-action.popover";
import { useEditRatelimits } from "@/hooks/use-edit-ratelimits";
import { usePersistedForm } from "@/hooks/use-persisted-form";
import type { RatelimitFormValues } from "@/lib/schemas/ratelimit";
import { ratelimitSchema } from "@/lib/schemas/ratelimit";
import type { KeyDetails } from "@/lib/trpc/routers/api/keys/query-api-keys/schema";
import { zodResolver } from "@hookform/resolvers/zod";
import { Button, DialogContainer } from "@unkey/ui";
import { useEffect } from "react";
import { FormProvider } from "react-hook-form";
import { KeyInfo } from "../key-info";
import { getKeyRatelimitsDefaults } from "./utils";

const EDIT_RATELIMITS_FORM_STORAGE_KEY = "unkey_edit_ratelimits_form_state";

type EditRatelimitsProps = {
  keyDetails: KeyDetails;
} & ActionComponentProps;

export const EditRatelimits = ({ keyDetails, isOpen, onClose }: EditRatelimitsProps) => {
  const methods = usePersistedForm<RatelimitFormValues>(
    `${EDIT_RATELIMITS_FORM_STORAGE_KEY}_${keyDetails.id}`,
    {
      resolver: zodResolver(ratelimitSchema),
      mode: "onChange",
      shouldFocusError: true,
      shouldUnregister: true,
      defaultValues: getKeyRatelimitsDefaults(keyDetails),
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

  const key = useEditRatelimits("key", () => {
    reset(getKeyRatelimitsDefaults(keyDetails));
    clearPersistedData();
    onClose();
  });

  const onSubmit = async (data: RatelimitFormValues) => {
    try {
      await key.mutateAsync({
        keyId: keyDetails.id,
        ratelimit: data.ratelimit,
      });
    } catch {
      // `useEditRatelimits` already shows a toast, but we still need to
      // prevent unhandled rejection noise in the console.
    }
  };

  return (
    <FormProvider {...methods}>
      <form id="edit-remaining-uses-form" onSubmit={handleSubmit(onSubmit)}>
        <DialogContainer
          isOpen={isOpen}
          subTitle="Control how often this key can be used"
          onOpenChange={() => {
            saveCurrentValues();
            onClose();
          }}
          title="Edit ratelimits"
          className="flex flex-col"
          contentClassName="flex flex-col flex-1 min-h-0"
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
                Update ratelimit
              </Button>
              <div className="text-gray-9 text-xs">Changes will be applied immediately</div>
            </div>
          }
        >
          {/* Scrollable body container */}
          <div className="flex-1 overflow-y-auto min-h-0 scrollbar-hide gap-4 flex flex-col">
            <KeyInfo keyDetails={keyDetails} />
            <div className="py-1 my-2">
              <div className="h-[1px] bg-grayA-3 w-full" />
            </div>
            <div className="[&>*:first-child]:p-0">
              <RatelimitSetup />
            </div>
          </div>
        </DialogContainer>
      </form>
    </FormProvider>
  );
};
