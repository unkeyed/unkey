import { KeyCreatedSuccessDialog } from "@/app/(app)/[workspaceSlug]/apis/[apiId]/_components/create-key/components/key-created-success-dialog";
import type { ActionComponentProps } from "@/components/logs/table-action.popover";
import { zodResolver } from "@hookform/resolvers/zod";
import { Button, ConfirmPopover, DialogContainer, FormCheckbox, FormSelect } from "@unkey/ui";
import { type ReactNode, useRef, useState } from "react";
import { Controller, FormProvider, useForm } from "react-hook-form";
import { z } from "zod";
import {
  DEFAULT_GRACE_PERIOD,
  GRACE_PERIOD_OPTIONS,
  type GracePeriodMs,
} from "./rotate-key.constants";

type RotatedKeyData = { id: string; key: string; name?: string };

type RotateInput = { keyId: string; expiration: GracePeriodMs };

type RotateMutation = {
  mutateAsync: (input: RotateInput) => Promise<{ keyId: string; key: string; name?: string }>;
};

export type RotateKeyDialogProps = {
  /** The id of the key being rotated. */
  keyId: string;
  /** Rendered above the form to show which key is being rotated. */
  info: ReactNode;
  /** Mutation that performs the rotation and surfaces its own error toasts. */
  mutation: RotateMutation;
  /**
   * Called with the rotated key after the user closes the success dialog. Use
   * this to invalidate any cached lists. Invalidating mid-flow remounts the
   * triggering row and wipes the success dialog before the user can copy the
   * secret.
   */
  onRotated?: (rotatedKey: RotatedKeyData) => void;
  /**
   * Optional builder for the "See key details" URL on the success dialog. When
   * omitted, the success dialog falls back to its default api-keys URL.
   */
  detailsHref?: (rotatedKey: RotatedKeyData) => string;
  /** apiId for the api-keys details URL fallback. Omit for root keys. */
  apiId?: string;
  /** keyspaceId for the api-keys details URL fallback. Omit for root keys. */
  keyspaceId?: string | null;
  /** "key" or "root key". Drives all user-facing copy. */
  resourceLabel: "key" | "root key";
  /** Unique form id; must not collide with other forms on the page. */
  formId: string;
} & ActionComponentProps;

export const RotateKeyDialog = ({
  keyId,
  info,
  mutation,
  onRotated,
  detailsHref,
  apiId,
  keyspaceId,
  resourceLabel,
  formId,
  isOpen,
  onClose,
}: RotateKeyDialogProps) => {
  const [isConfirmPopoverOpen, setIsConfirmPopoverOpen] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const [rotatedKeyData, setRotatedKeyData] = useState<RotatedKeyData | null>(null);
  const rotateButtonRef = useRef<HTMLButtonElement>(null);

  const schema = z.object({
    gracePeriod: z.string(),
    confirmRotation: z.boolean().refine((val) => val === true, {
      error: `Please confirm that you want to rotate this ${resourceLabel}`,
    }),
  });

  const methods = useForm<z.infer<typeof schema>>({
    resolver: zodResolver(schema),
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

  const handleDialogOpenChange = (open: boolean) => {
    if (open) {
      return;
    }
    // The confirm popover and success dialog each own their own close path;
    // ignore the form-level onOpenChange while either is active.
    if (isConfirmPopoverOpen || rotatedKeyData) {
      return;
    }
    onClose();
  };

  const performRotation = async () => {
    try {
      setIsLoading(true);
      const result = await mutation.mutateAsync({
        keyId,
        // GRACE_PERIOD_OPTIONS and GracePeriodMs are derived from the same
        // GRACE_PERIODS table, so any value the FormSelect can produce is
        // a valid GracePeriodMs at runtime. The cast bridges the form's
        // string typing without losing that guarantee.
        expiration: Number(gracePeriod) as GracePeriodMs,
      });
      setRotatedKeyData({ id: result.keyId, key: result.key, name: result.name });
    } catch {
      // The mutation hook surfaces its own toast.
    } finally {
      setIsLoading(false);
    }
  };

  const handleSuccessDialogClose = () => {
    const data = rotatedKeyData;
    setRotatedKeyData(null);
    if (data) {
      onRotated?.(data);
    }
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
        variant="rotated"
        resourceLabel={resourceLabel}
        detailsHref={detailsHref?.(rotatedKeyData)}
      />
    );
  }

  const titleCase = resourceLabel === "root key" ? "Rotate root key" : "Rotate key";
  const confirmTitle = `Confirm ${resourceLabel} rotation`;
  const confirmDescription = `A new ${resourceLabel} will be generated now. The current ${resourceLabel} will be revoked after the grace period you selected.`;
  const subTitle =
    resourceLabel === "root key"
      ? "Generate a fresh root key while preserving this root key's permissions"
      : "Generate a fresh key while preserving this key's configuration";
  const gracePeriodDescription = `How long the current ${resourceLabel} stays valid after rotation.`;

  return (
    <>
      <FormProvider {...methods}>
        <form id={formId}>
          <DialogContainer
            isOpen={isOpen}
            subTitle={subTitle}
            onOpenChange={handleDialogOpenChange}
            title={titleCase}
            footer={
              <div className="w-full flex flex-col gap-2 items-center justify-center">
                <Button
                  type="button"
                  form={formId}
                  variant="primary"
                  color="danger"
                  size="xlg"
                  className="w-full rounded-lg"
                  disabled={!confirmRotation || isLoading}
                  loading={isLoading}
                  onClick={() => setIsConfirmPopoverOpen(true)}
                  ref={rotateButtonRef}
                >
                  {titleCase}
                </Button>
                <div className="text-gray-9 text-xs">You will see the new secret only once</div>
              </div>
            }
          >
            {info}
            <Controller
              name="gracePeriod"
              control={control}
              render={({ field }) => (
                <FormSelect
                  label="Grace period"
                  description={gracePeriodDescription}
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
                  id={`${formId}-confirm`}
                  color="danger"
                  size="lg"
                  onCheckedChange={field.onChange}
                  requirement="required"
                  label={`I understand this will generate a new ${resourceLabel} and revoke the current one.`}
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
        onConfirm={performRotation}
        triggerRef={rotateButtonRef}
        title={confirmTitle}
        description={confirmDescription}
        confirmButtonText={titleCase}
        cancelButtonText="Cancel"
        variant="danger"
      />
    </>
  );
};
