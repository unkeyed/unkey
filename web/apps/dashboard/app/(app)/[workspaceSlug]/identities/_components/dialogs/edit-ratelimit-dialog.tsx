"use client";

import { RatelimitSetup } from "@/components/dashboard/ratelimits/ratelimit-setup";
import type { ActionComponentProps } from "@/components/logs/table-action.popover";
import { useEditIdentityRatelimits } from "@/hooks/use-edit-ratelimits";
import { usePersistedForm } from "@/hooks/use-persisted-form";
import type { RatelimitFormValues } from "@/lib/schemas/ratelimit";
import { ratelimitSchema } from "@/lib/schemas/ratelimit";
import type { DiscriminatedUnionResolver } from "@/lib/schemas/resolver-types";
import type { IdentityResponseSchema } from "@/lib/trpc/routers/identity/query";
import { zodResolver } from "@hookform/resolvers/zod";
import { Button, DialogContainer } from "@unkey/ui";
import { type FC, useEffect, useId } from "react";
import { FormProvider } from "react-hook-form";
import type { z } from "zod";
import { IdentityInfo } from "./identity-info";

type Identity = z.infer<typeof IdentityResponseSchema>;

type EditRatelimitDialogProps = { identity: Identity } & ActionComponentProps;

const EDIT_RATELIMITS_FORM_STORAGE_KEY = "unkey_edit_identity_ratelimits_form_state";

const getIdentityRatelimitsDefaults = (identity: Identity) => {
  const hasRatelimits = identity.ratelimits && identity.ratelimits.length > 0;

  return {
    ratelimit: hasRatelimits
      ? ({
          enabled: true as const,
          data: identity.ratelimits.map((rl) => ({
            id: rl.id,
            name: rl.name,
            limit: rl.limit,
            refillInterval: rl.duration,
            autoApply: rl.autoApply,
          })),
        } as const)
      : ({ enabled: false as const } as const),
  };
};

export const EditRatelimitDialog: FC<EditRatelimitDialogProps> = ({
  identity,
  isOpen,
  onClose,
}) => {
  const formId = useId();

  const methods = usePersistedForm<RatelimitFormValues>(
    `${EDIT_RATELIMITS_FORM_STORAGE_KEY}_${identity.id}`,
    {
      resolver: zodResolver(ratelimitSchema) as DiscriminatedUnionResolver<typeof ratelimitSchema>,
      mode: "onChange",
      shouldFocusError: true,
      shouldUnregister: true,
      defaultValues: getIdentityRatelimitsDefaults(identity),
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

  useEffect(() => {
    if (isOpen) {
      loadSavedValues();
    }
  }, [isOpen, loadSavedValues]);

  const updateRatelimit = useEditIdentityRatelimits(() => {
    reset(getIdentityRatelimitsDefaults(identity));
    clearPersistedData();
    onClose();
  });

  const onSubmit = async (data: RatelimitFormValues) => {
    try {
      await updateRatelimit.mutateAsync({
        identityId: identity.id,
        ratelimit: data.ratelimit,
      });
    } catch {
      // `useEditIdentityRatelimits` already shows a toast, but we still
      // need to prevent unhandled rejection noise in the console.
    }
  };

  return (
    <FormProvider {...methods}>
      <form id={formId} onSubmit={handleSubmit(onSubmit)}>
        <DialogContainer
          isOpen={isOpen}
          onOpenChange={(o) => {
            if (!o) {
              saveCurrentValues();
              onClose();
            }
          }}
          title="Edit ratelimit"
          subTitle="Control how often this identity can be used"
          className="flex flex-col"
          contentClassName="flex flex-col flex-1 min-h-0"
          footer={
            <div className="w-full flex flex-col gap-2 items-center justify-center">
              <Button
                type="submit"
                form={formId}
                variant="primary"
                size="xlg"
                className="w-full rounded-lg"
                disabled={!isValid || isSubmitting}
                loading={updateRatelimit.isLoading}
              >
                Update ratelimit
              </Button>
              <div className="text-gray-9 text-xs">Changes will be applied immediately</div>
            </div>
          }
        >
          {/* Scrollable body container */}
          <div className="flex-1 overflow-y-auto min-h-0 scrollbar-hide gap-4 flex flex-col">
            <IdentityInfo identity={identity} />
            <div className="py-1 my-2">
              <div className="h-px bg-grayA-3 w-full" />
            </div>
            <div className="[&>*:first-child]:p-0">
              <RatelimitSetup entityType="identity" />
            </div>
          </div>
        </DialogContainer>
      </form>
    </FormProvider>
  );
};
