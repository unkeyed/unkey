import { DialogContainer } from "@/components/dialog-container";
import type { KeyDetails } from "@/lib/trpc/routers/api/keys/query-api-keys/schema";
import { zodResolver } from "@hookform/resolvers/zod";
import { TriangleWarning2 } from "@unkey/icons";
import { Button, FormCheckbox } from "@unkey/ui";
import { Controller, FormProvider, useForm } from "react-hook-form";
import { z } from "zod";
import type { ActionComponentProps } from "../keys-table-action.popover";
import { useDeleteKey } from "./hooks/use-delete-key";
import { KeyInfo } from "./key-info";

const deleteKeyFormSchema = z.object({
  confirmDeletion: z.boolean().refine((val) => val === true, {
    message: "Please confirm that you want to permanently delete this key",
  }),
});

type DeleteKeyFormValues = z.infer<typeof deleteKeyFormSchema>;

type DeleteKeyProps = { keyDetails: KeyDetails } & ActionComponentProps;

export const DeleteKey = ({ keyDetails, isOpen, onClose }: DeleteKeyProps) => {
  const methods = useForm<DeleteKeyFormValues>({
    resolver: zodResolver(deleteKeyFormSchema),
    mode: "onChange",
    shouldFocusError: true,
    shouldUnregister: true,
    defaultValues: {
      confirmDeletion: false,
    },
  });

  const {
    handleSubmit,
    formState: { isSubmitting, errors },
    control,
    watch,
  } = methods;

  const confirmDeletion = watch("confirmDeletion");

  const deleteKey = useDeleteKey(() => {
    onClose();
  });

  const onSubmit = async () => {
    try {
      await deleteKey.mutateAsync({
        keyIds: [keyDetails.id],
      });
    } catch {
      // `useDeleteKey` already shows a toast, but we still need to
      // prevent unhandled‐rejection noise in the console.
    }
  };

  return (
    <FormProvider {...methods}>
      <form id="delete-key-form" onSubmit={handleSubmit(onSubmit)}>
        <DialogContainer
          isOpen={isOpen}
          subTitle="Permanently remove this key and its data"
          onOpenChange={onClose}
          title="Delete key"
          footer={
            <div className="w-full flex flex-col gap-2 items-center justify-center">
              <Button
                type="submit"
                form="delete-key-form"
                variant="primary"
                color="danger"
                size="xlg"
                className="w-full rounded-lg"
                disabled={!confirmDeletion || isSubmitting}
                loading={isSubmitting}
              >
                Delete key
              </Button>
              <div className="text-gray-9 text-xs">
                This key will be permanently deleted immediately
              </div>
            </div>
          }
        >
          <KeyInfo keyDetails={keyDetails} />
          <div className="py-1 my-2">
            <div className="h-[1px] bg-grayA-3 w-full" />
          </div>
          <div className="rounded-xl bg-errorA-2 dark:bg-black border border-errorA-3 flex items-center gap-4 px-[22px] py-6">
            <div className="bg-error-9 size-8 rounded-full flex items-center justify-center flex-shrink-0">
              <TriangleWarning2 size="sm-regular" className="text-white" />
            </div>
            <div className="text-error-12 text-[13px] leading-6">
              <span className="font-medium">Warning:</span> deleting this key will remove all
              associated data and metadata. This action cannot be undone. Any verification,
              tracking, and historical usage tied to this key will be permanently lost.
            </div>
          </div>
          <Controller
            name="confirmDeletion"
            control={control}
            render={({ field }) => (
              <FormCheckbox
                id="confirm-deletion"
                className="mt-2"
                color="danger"
                size="md"
                checked={field.value}
                onCheckedChange={field.onChange}
                label="I understand this will permanently delete the key and all its associated data"
                error={errors.confirmDeletion?.message}
              />
            )}
          />
        </DialogContainer>
      </form>
    </FormProvider>
  );
};
