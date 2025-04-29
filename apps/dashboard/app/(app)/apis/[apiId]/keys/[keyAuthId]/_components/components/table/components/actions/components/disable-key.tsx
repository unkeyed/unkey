import { RatelimitOverviewTooltip } from "@/app/(app)/ratelimits/[namespaceId]/_overview/components/table/components/ratelimit-overview-tooltip";
import { DialogContainer } from "@/components/dialog-container";
import type { KeyDetails } from "@/lib/trpc/routers/api/keys/query-api-keys/schema";
import { zodResolver } from "@hookform/resolvers/zod";
import { Key2 } from "@unkey/icons";
import { Button, FormCheckbox } from "@unkey/ui";
import { Controller, FormProvider, useForm } from "react-hook-form";
import { z } from "zod";
import type { ActionComponentProps } from "../keys-table-action.popover";
import { useUpdateKeyStatus } from "./hooks/use-update-key-status";

const updateKeyStatusFormSchema = z.object({
  confirmStatusChange: z.boolean().refine((val) => val === true, {
    message: "Please confirm that you want to change this key's status",
  }),
});

type UpdateKeyStatusFormValues = z.infer<typeof updateKeyStatusFormSchema>;
type UpdateKeyStatusProps = { keyDetails: KeyDetails } & ActionComponentProps;

export const UpdateKeyStatus = ({ keyDetails, isOpen, onClose }: UpdateKeyStatusProps) => {
  const isEnabling = !keyDetails.enabled;
  const action = isEnabling ? "Enable" : "Disable";
  const actionColor = isEnabling ? "default" : "danger";

  const methods = useForm<UpdateKeyStatusFormValues>({
    resolver: zodResolver(updateKeyStatusFormSchema),
    mode: "onChange",
    shouldFocusError: true,
    shouldUnregister: true,
    defaultValues: {
      confirmStatusChange: true,
    },
  });

  const {
    handleSubmit,
    formState: { isSubmitting, errors },
    control,
    watch,
  } = methods;

  const confirmStatusChange = watch("confirmStatusChange");

  const updateKeyStatus = useUpdateKeyStatus(() => {
    onClose();
  });

  const onSubmit = async () => {
    try {
      await updateKeyStatus.mutateAsync({
        keyId: keyDetails.id,
        enabled: isEnabling,
      });
    } catch {
      // `useUpdateKeyStatus` already shows a toast, but we still need to
      // prevent unhandled‐rejection noise in the console.
    }
  };

  return (
    <FormProvider {...methods}>
      <form id="update-key-status-form" onSubmit={handleSubmit(onSubmit)}>
        <DialogContainer
          isOpen={isOpen}
          subTitle={
            isEnabling
              ? "Enable this key to allow verification requests"
              : "Disable this key to block verification requests"
          }
          onOpenChange={onClose}
          title={`${action} key`}
          footer={
            <div className="w-full flex flex-col gap-2 items-center justify-center">
              <Button
                type="submit"
                form="update-key-status-form"
                variant="primary"
                color={actionColor}
                size="xlg"
                className="w-full rounded-lg"
                disabled={!confirmStatusChange || isSubmitting}
                loading={isSubmitting}
              >
                {`${action} key`}
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
            name="confirmStatusChange"
            control={control}
            render={({ field }) => (
              <FormCheckbox
                id="confirm-status-change"
                color={actionColor}
                size="md"
                checked={field.value}
                onCheckedChange={field.onChange}
                label={
                  isEnabling
                    ? "I want to enable this key and allow verification"
                    : "I want to disable this key and stop all verification"
                }
                error={errors.confirmStatusChange?.message}
              />
            )}
          />
        </DialogContainer>
      </form>
    </FormProvider>
  );
};
