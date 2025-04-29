import { RatelimitOverviewTooltip } from "@/app/(app)/ratelimits/[namespaceId]/_overview/components/table/components/ratelimit-overview-tooltip";
import { DialogContainer } from "@/components/dialog-container";
import type { KeyDetails } from "@/lib/trpc/routers/api/keys/query-api-keys/schema";
import { zodResolver } from "@hookform/resolvers/zod";
import { Key2 } from "@unkey/icons";
import { Button, FormCheckbox } from "@unkey/ui";
import { Controller, FormProvider, useForm } from "react-hook-form";
import { z } from "zod";
import type { ActionComponentProps } from "../keys-table-action.popover";

const disableKeyFormSchema = z.object({
  confirmDisable: z.boolean().refine((val) => val === true, {
    message: "Please confirm that you want to disable this key",
  }),
});

type DisableKeyFormValues = z.infer<typeof disableKeyFormSchema>;
type DisableKeyProps = { keyDetails: KeyDetails } & ActionComponentProps;

export const DisableKey = ({ keyDetails, isOpen, onClose }: DisableKeyProps) => {
  const methods = useForm<DisableKeyFormValues>({
    resolver: zodResolver(disableKeyFormSchema),
    mode: "onChange",
    shouldFocusError: true,
    shouldUnregister: true,
    defaultValues: {
      confirmDisable: false,
    },
  });

  const {
    handleSubmit,
    formState: { isSubmitting, errors },
    control,
    watch,
  } = methods;

  // Watch confirmDisable value to enable/disable the submit button
  const confirmDisable = watch("confirmDisable");

  const onSubmit = async () => {
    try {
      // await key.mutateAsync({ ...data, keyId: keyDetails.id });
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
          subTitle="Disable this key to block verification requests"
          onOpenChange={onClose}
          title="Disable key"
          footer={
            <div className="w-full flex flex-col gap-2 items-center justify-center">
              <Button
                type="submit"
                form="edit-key-name-form"
                variant="primary"
                color="danger"
                size="xlg"
                className="w-full rounded-lg"
                disabled={!confirmDisable || isSubmitting}
              >
                {isSubmitting ? "Saving..." : "Disable key"}
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

          <Controller
            name="confirmDisable"
            control={control}
            render={({ field }) => (
              <FormCheckbox
                id="terms"
                color="danger"
                size="md"
                checked={field.value}
                onCheckedChange={field.onChange}
                label="I want to disable this key and stop all verification"
                error={errors.confirmDisable?.message}
              />
            )}
          />
        </DialogContainer>
      </form>
    </FormProvider>
  );
};
