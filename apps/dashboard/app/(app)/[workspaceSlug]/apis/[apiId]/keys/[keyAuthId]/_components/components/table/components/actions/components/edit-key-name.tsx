import { nameSchema } from "@/app/(app)/[workspaceSlug]/apis/[apiId]/_components/create-key/create-key.schema";
import type { ActionComponentProps } from "@/components/logs/table-action.popover";
import { usePersistedForm } from "@/hooks/use-persisted-form";
import type { KeyDetails } from "@/lib/trpc/routers/api/keys/query-api-keys/schema";
import { zodResolver } from "@hookform/resolvers/zod";
import { Button, DialogContainer, FormInput } from "@unkey/ui";
import { useEffect } from "react";
import { FormProvider } from "react-hook-form";
import { z } from "zod";
import { useEditKeyName } from "./hooks/use-edit-key";
import { KeyInfo } from "./key-info";

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
    `${EDIT_NAME_FORM_STORAGE_KEY}_${keyDetails.id}`,
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
                loading={key.isPending}
              >
                {isSubmitting ? "Saving..." : "Update key name"}
              </Button>
              <div className="text-gray-9 text-xs">Changes will be applied immediately</div>
            </div>
          }
        >
          <KeyInfo keyDetails={keyDetails} />
          <div className="py-1 my-2">
            <div className="h-[1px] bg-grayA-3 w-full" />
          </div>
          <div className="space-y-4">
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
