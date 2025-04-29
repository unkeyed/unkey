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
import { useEditKeyName } from "./hooks/use-edit-key";

const editNameFormSchema = z
  .object({
    name: nameSchema,
    //Hidden field. Required for comparison
    originalName: z.string().optional().default(""),
  })
  .superRefine((data, ctx) => {
    const normalizedNewName = (data.name || "").trim();
    const normalizedOriginalName = (data.originalName || "").trim();

    if (normalizedNewName === normalizedOriginalName && normalizedNewName !== "") {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        message: "New name must be different from the current name",
        path: ["name"],
      });
    }
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
        originalName: keyDetails.name || "",
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
    clearPersistedData,
    reset,
  } = methods;

  // Load saved values when the dialog opens
  useEffect(() => {
    if (isOpen) {
      loadSavedValues();
    }
  }, [isOpen, loadSavedValues]);

  const key = useEditKeyName(() => {
    clearPersistedData();
    reset({
      name: keyDetails.name || "",
      originalName: keyDetails.name || "",
    });
    onClose();
  });

  const onSubmit = async (data: EditNameFormValues) => {
    try {
      await key.mutateAsync({ ...data, keyId: keyDetails.id });
    } catch {
      // `useEditKeyName` already shows a toast, but we still need to
      // prevent unhandled‐rejection noise in the console.
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
          title="Edit key name"
          footer={
            <div className="w-full flex flex-col gap-2 items-center justify-center">
              <Button
                type="submit"
                form="edit-key-name-form"
                variant="primary"
                size="xlg"
                className="w-full rounded-lg"
                disabled={!isValid || isSubmitting}
                loading={key.isLoading}
              >
                {isSubmitting ? "Saving..." : "Update key name"}
              </Button>
              <div className="text-gray-9 text-xs">Changes will be applied immediately</div>
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
            <FormInput
              hidden
              className="hidden"
              defaultValue={keyDetails.name ?? ""}
              {...register("originalName")}
            />
          </div>
        </DialogContainer>
      </form>
    </FormProvider>
  );
};
