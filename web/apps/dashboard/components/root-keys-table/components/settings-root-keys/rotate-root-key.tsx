import { KeyCreatedSuccessDialog } from "@/app/(app)/[workspaceSlug]/apis/[apiId]/_components/create-key/components/key-created-success-dialog";
import {
  DEFAULT_GRACE_PERIOD,
  GRACE_PERIOD_OPTIONS,
} from "@/components/api-keys-table/components/actions/components/rotate-key/rotate-key.constants";
import type { ActionComponentProps } from "@/components/logs/table-action.popover";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { trpc } from "@/lib/trpc/client";
import type { RootKey } from "@/lib/trpc/routers/settings/root-keys/query";
import { zodResolver } from "@hookform/resolvers/zod";
import { Button, ConfirmPopover, DialogContainer, FormCheckbox, FormSelect } from "@unkey/ui";
import { useRef, useState } from "react";
import { Controller, FormProvider, useForm } from "react-hook-form";
import { z } from "zod";
import { useRotateRootKey } from "../../hooks/use-rotate-root-key";
import { RootKeyInfo } from "./root-key-info";

const rotateRootKeyFormSchema = z.object({
  gracePeriod: z.string(),
  confirmRotation: z.boolean().refine((val) => val === true, {
    error: "Please confirm that you want to rotate this root key",
  }),
});

type RotateRootKeyFormValues = z.infer<typeof rotateRootKeyFormSchema>;

type RotateRootKeyProps = { rootKeyDetails: RootKey } & ActionComponentProps;

type RotatedKeyData = { key: string; id: string; name?: string };

export const RotateRootKey = ({ rootKeyDetails, isOpen, onClose }: RotateRootKeyProps) => {
  const workspace = useWorkspaceNavigation();
  const trpcUtils = trpc.useUtils();
  const [isConfirmPopoverOpen, setIsConfirmPopoverOpen] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const [rotatedKeyData, setRotatedKeyData] = useState<RotatedKeyData | null>(null);
  const rotateButtonRef = useRef<HTMLButtonElement>(null);

  const methods = useForm<RotateRootKeyFormValues>({
    resolver: zodResolver(rotateRootKeyFormSchema),
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

  const rotateRootKey = useRotateRootKey((data) => {
    setRotatedKeyData({ id: data.keyId, key: data.key, name: data.name });
  });

  const handleDialogOpenChange = (open: boolean) => {
    if (open) {
      return;
    }
    if (isConfirmPopoverOpen) {
      return;
    }
    if (rotatedKeyData) {
      return;
    }
    onClose();
  };

  const handleRotateButtonClick = () => {
    setIsConfirmPopoverOpen(true);
  };

  const performRootKeyRotation = async () => {
    try {
      setIsLoading(true);
      await rotateRootKey.mutateAsync({
        keyId: rootKeyDetails.id,
        expiration: Number(gracePeriod),
      });
    } catch {
      // `useRotateRootKey` already surfaces a toast
    } finally {
      setIsLoading(false);
    }
  };

  const handleSuccessDialogClose = () => {
    setRotatedKeyData(null);
    // Invalidate after the user has seen the new secret. Invalidating
    // mid-flow refetches the list, which remounts this component via
    // the table cell and wipes the success dialog.
    trpcUtils.settings.rootKeys.query.invalidate();
    onClose();
  };

  if (rotatedKeyData) {
    return (
      <KeyCreatedSuccessDialog
        apiId=""
        isOpen
        onClose={handleSuccessDialogClose}
        keyData={rotatedKeyData}
        variant="rerolled"
        detailsHref={`/${workspace.slug}/settings/root-keys/${rotatedKeyData.id}`}
      />
    );
  }

  return (
    <>
      <FormProvider {...methods}>
        <form id="rotate-root-key-form">
          <DialogContainer
            isOpen={isOpen}
            subTitle="Generate a fresh secret while preserving this root key's permissions"
            onOpenChange={handleDialogOpenChange}
            title="Rotate root key"
            footer={
              <div className="w-full flex flex-col gap-2 items-center justify-center">
                <Button
                  type="button"
                  form="rotate-root-key-form"
                  variant="primary"
                  color="danger"
                  size="xlg"
                  className="w-full rounded-lg"
                  disabled={!confirmRotation || isLoading}
                  loading={isLoading}
                  onClick={handleRotateButtonClick}
                  ref={rotateButtonRef}
                >
                  Rotate root key
                </Button>
                <div className="text-gray-9 text-xs">You will see the new secret only once</div>
              </div>
            }
          >
            <RootKeyInfo rootKeyDetails={rootKeyDetails} />
            <Controller
              name="gracePeriod"
              control={control}
              render={({ field }) => (
                <FormSelect
                  label="Grace period"
                  description="How long the current root key stays valid after rotation."
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
                  id="confirm-root-rotation"
                  color="danger"
                  size="lg"
                  onCheckedChange={field.onChange}
                  requirement="required"
                  label="I understand this will generate a new secret and revoke the current one."
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
        onConfirm={performRootKeyRotation}
        triggerRef={rotateButtonRef}
        title="Confirm root key rotation"
        description="A new secret will be generated now. The current root key will be revoked after the grace period you selected."
        confirmButtonText="Rotate root key"
        cancelButtonText="Cancel"
        variant="warning"
      />
    </>
  );
};
