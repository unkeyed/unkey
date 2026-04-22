import { KeyCreatedSuccessDialog } from "@/app/(app)/[workspaceSlug]/apis/[apiId]/_components/create-key/components/key-created-success-dialog";
import type { ActionComponentProps } from "@/components/logs/table-action.popover";
import { trpc } from "@/lib/trpc/client";
import type { KeyDetails } from "@/lib/trpc/routers/api/keys/query-api-keys/schema";
import { zodResolver } from "@hookform/resolvers/zod";
import { Button, ConfirmPopover, DialogContainer, FormCheckbox, FormSelect } from "@unkey/ui";
import { useRef, useState } from "react";
import { Controller, FormProvider, useForm } from "react-hook-form";
import { z } from "zod";
import { KeyInfo } from "../key-info";
import { useRotateKey } from "./hooks/use-rotate-key";
import { DEFAULT_GRACE_PERIOD, GRACE_PERIOD_OPTIONS } from "./rotate-key.constants";

const rotateKeyFormSchema = z.object({
  gracePeriod: z.string(),
  confirmRotation: z.boolean().refine((val) => val === true, {
    error: "Please confirm that you want to rotate this key",
  }),
});

type RotateKeyFormValues = z.infer<typeof rotateKeyFormSchema>;

type RotateKeyProps = {
  keyDetails: KeyDetails;
  apiId: string;
  keyspaceId?: string | null;
} & ActionComponentProps;

type RotatedKeyData = { key: string; id: string; name?: string };

export const RotateKey = ({ keyDetails, apiId, keyspaceId, isOpen, onClose }: RotateKeyProps) => {
  const trpcUtils = trpc.useUtils();
  const [isConfirmPopoverOpen, setIsConfirmPopoverOpen] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const [rotatedKeyData, setRotatedKeyData] = useState<RotatedKeyData | null>(null);
  const rotateButtonRef = useRef<HTMLButtonElement>(null);

  const methods = useForm<RotateKeyFormValues>({
    resolver: zodResolver(rotateKeyFormSchema),
    mode: "onChange",
    shouldFocusError: true,
    shouldUnregister: true,
    defaultValues: {
      gracePeriod: DEFAULT_GRACE_PERIOD,
      confirmRotation: false,
    },
  });

  const {
    formState: { errors },
    control,
    watch,
  } = methods;

  const confirmRotation = watch("confirmRotation");
  const gracePeriod = watch("gracePeriod");

  const rotateKey = useRotateKey((data) => {
    setRotatedKeyData({ id: data.keyId, key: data.key, name: data.name });
  });

  const handleDialogOpenChange = (open: boolean) => {
    if (open) {
      return;
    }
    // Confirm popover owns the close in that path.
    if (isConfirmPopoverOpen) {
      return;
    }
    // Form is closing because we're about to show the success dialog.
    if (rotatedKeyData) {
      return;
    }
    onClose();
  };

  const handleRotateButtonClick = () => {
    setIsConfirmPopoverOpen(true);
  };

  const performKeyRotation = async () => {
    try {
      setIsLoading(true);
      await rotateKey.mutateAsync({
        keyId: keyDetails.id,
        expiration: Number(gracePeriod),
      });
    } catch {
      // `useRotateKey` already surfaces a toast
    } finally {
      setIsLoading(false);
    }
  };

  const handleSuccessDialogClose = () => {
    setRotatedKeyData(null);
    // Invalidate after the user has seen the new secret. Invalidating
    // mid-flow refetches the keys list, which remounts this component
    // via the table cell and wipes the success dialog.
    trpcUtils.api.keys.list.invalidate();
    trpcUtils.api.overview.keyCount.invalidate();
    onClose();
  };

  if (rotatedKeyData) {
    return (
      <KeyCreatedSuccessDialog
        apiId={apiId}
        keyspaceId={keyspaceId}
        isOpen
        onClose={handleSuccessDialogClose}
        keyData={rotatedKeyData}
        variant="rerolled"
      />
    );
  }

  return (
    <>
      <FormProvider {...methods}>
        <form id="rotate-key-form">
          <DialogContainer
            isOpen={isOpen}
            subTitle="Generate a fresh key while preserving this key's configuration"
            onOpenChange={handleDialogOpenChange}
            title="Rotate key"
            footer={
              <div className="w-full flex flex-col gap-2 items-center justify-center">
                <Button
                  type="button"
                  form="rotate-key-form"
                  variant="primary"
                  color="danger"
                  size="xlg"
                  className="w-full rounded-lg"
                  disabled={!confirmRotation || isLoading}
                  loading={isLoading}
                  onClick={handleRotateButtonClick}
                  ref={rotateButtonRef}
                >
                  Rotate key
                </Button>
                <div className="text-gray-9 text-xs">You will see the new secret only once</div>
              </div>
            }
          >
            <KeyInfo keyDetails={keyDetails} />
            <Controller
              name="gracePeriod"
              control={control}
              render={({ field }) => (
                <FormSelect
                  label="Grace period"
                  description="How long the current key stays valid after rotation."
                  options={GRACE_PERIOD_OPTIONS}
                  value={field.value}
                  onValueChange={field.onChange}
                  error={errors.gracePeriod?.message}
                />
              )}
            />
            <Controller
              name="confirmRotation"
              control={control}
              render={({ field }) => (
                <FormCheckbox
                  id="confirm-rotation"
                  color="danger"
                  size="lg"
                  onCheckedChange={field.onChange}
                  requirement="required"
                  label="I understand this will generate a new key and revoke the current one."
                  error={errors.confirmRotation?.message}
                />
              )}
            />
          </DialogContainer>
        </form>
      </FormProvider>
      <ConfirmPopover
        isOpen={isConfirmPopoverOpen}
        onOpenChange={setIsConfirmPopoverOpen}
        onConfirm={performKeyRotation}
        triggerRef={rotateButtonRef}
        title="Confirm key rotation"
        description="A new secret will be generated now. The current key will be revoked after the grace period you selected."
        confirmButtonText="Rotate key"
        cancelButtonText="Cancel"
        variant="warning"
      />
    </>
  );
};
