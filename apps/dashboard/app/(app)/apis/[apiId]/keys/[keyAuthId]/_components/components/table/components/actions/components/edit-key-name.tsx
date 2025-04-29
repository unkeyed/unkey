import { nameSchema } from "@/app/(app)/apis/[apiId]/_components/create-key/create-key.schema";
import { RatelimitOverviewTooltip } from "@/app/(app)/ratelimits/[namespaceId]/_overview/components/table/components/ratelimit-overview-tooltip";
import { DialogContainer } from "@/components/dialog-container";
import { usePersistedForm } from "@/hooks/use-persisted-form";
import type { KeyDetails } from "@/lib/trpc/routers/api/keys/query-api-keys/schema";
import { zodResolver } from "@hookform/resolvers/zod";
import { Key2 } from "@unkey/icons";
import { Button, FormInput } from "@unkey/ui";
import { useEffect } from "react";
import { FormProvider } from "react-hook-form";
import { z } from "zod";
import type { ActionComponentProps } from "../keys-table-action.popover";

const editNameFormSchema = z.object({
  name: nameSchema,
});
const EDIT_NAME_FORM_STORAGE_KEY = "unkey_edit_name_form_state";

type EditNameFormValues = z.infer<typeof editNameFormSchema>;
type EditKeyNameProps = { keyDetails: KeyDetails } & ActionComponentProps;

export const EditKeyName = ({ keyDetails, isOpen, onClose }: EditKeyNameProps) => {
  const methods = usePersistedForm<EditNameFormValues>(
    EDIT_NAME_FORM_STORAGE_KEY,
    {
      resolver: zodResolver(editNameFormSchema),
      mode: "onChange",
      shouldFocusError: true,
      shouldUnregister: true,
      defaultValues: {
        name: keyDetails.name || "",
      },
    },
    "memory",
  );

  const {
    handleSubmit,
    formState: { isSubmitting, errors, isValid },
    register,
    loadSavedValues,
    saveCurrentValues,
  } = methods;

  // Load saved values when the dialog opens
  useEffect(() => {
    if (isOpen) {
      loadSavedValues();
    }
  }, [isOpen, loadSavedValues]);

  const onSubmit = async (data: EditNameFormValues) => {
    try {
      // Add your API call here to update the key name
      // Example: await updateKeyName(key.id, data.name);
      console.log("Updating key name to:", data.name);

      // Close the dialog after successful submission
      onClose();
    } catch (error) {
      console.error("Failed to update key name:", error);
    }
  };

  return (
    <FormProvider {...methods}>
      <form id="edit-key-name-form" onSubmit={handleSubmit(onSubmit)}>
        <DialogContainer
          isOpen={isOpen}
          subTitle="Name this key for easy identification"
          onOpenChange={() => {
            saveCurrentValues();
            onClose();
          }}
          title="Create Namespace"
          footer={
            <div className="w-full flex flex-col gap-2 items-center justify-center">
              <Button
                type="submit"
                form="edit-key-name-form"
                variant="primary"
                size="xlg"
                className="w-full rounded-lg"
                disabled={!isValid || isSubmitting}
                // loading={key.isLoading}
              >
                {isSubmitting ? "Saving..." : "Save"}
              </Button>
            </div>
          }
        >
          <div className="flex gap-5 items-center bg-white dark:bg-black border border-grayA-5 rounded-xl py-5 pl-[18px] pr-[26px]">
            <div className="bg-grayA-5 text-gray-12 size-5 flex items-center justify-center rounded ">
              <Key2 size="sm-regular" />
            </div>
            <div className="flex flex-col gap-1">
              <div className="text-accent-12 text-xs font-mono">{keyDetails.id}</div>
              <RatelimitOverviewTooltip
                content={keyDetails.name}
                position={{ side: "bottom", align: "center" }}
                asChild
                disabled={!keyDetails.name}
              >
                <div className="text-accent-9 text-xs max-w-[160px] truncate">
                  {keyDetails.name ?? "Unnamed Key"}
                </div>
              </RatelimitOverviewTooltip>
            </div>
          </div>

          <div className="py-1 my-2">
            <div className="h-[1px] bg-grayA-3 w-full" />
          </div>
          <div className="space-y-4">
            <FormInput
              className="[&_input:first-of-type]:h-[36px]"
              placeholder="Key ID"
              label="Name"
              maxLength={256}
              readOnly
              defaultValue={keyDetails.id}
              description="An identifier for the API, used in some API calls."
              variant="default"
            />
            <FormInput
              className="[&_input:first-of-type]:h-[36px]"
              placeholder="Key Name"
              label="Name"
              defaultValue={keyDetails.name ?? ""}
              maxLength={256}
              description="Not customer-facing. Choose a name that is easy to recognize."
              error={errors.name?.message}
              variant="default"
              optional
              {...register("name")}
            />
          </div>
        </DialogContainer>
      </form>
    </FormProvider>
  );
};
